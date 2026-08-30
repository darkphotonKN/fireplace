# Learning Plan: Multi-Tier Hierarchy + Task Dependencies on Fireplace

A brief for an LLM mentor. Goal: take Fireplace's existing single-tier
nesting and rebuild it as a true unbounded hierarchy with task
dependencies, picking up production-grade Postgres patterns along the
way. This is a **learning exercise**, not a feature ship — present
trade-offs, ask which I'd pick given the access pattern, then walk me
through the implementation of the chosen path.

The repo's actual code may be a few commits ahead of what's described
here. Ask the operator to paste the latest of:

- `internal/checklistitems/{repository,service,handler,model}.go`
- `internal/models/entities.go`
- The most recent migration file in `migrations/`

---

## 1. Project context

**Stack:** Go 1.23 + Gin + sqlx + PostgreSQL, Next.js 15 frontend.

**Architecture:** handler → service → repository (interface DI). sqlx
with named queries. `errorutils.AnalyzeDBErr` wraps DB errors at the
repo layer. Service errors wrap `constants.ErrInvalidInput`,
`ErrNotFound`, `ErrForbidden` so handlers map cleanly to HTTP codes.

**Domain:** users own plans; plans contain `checklist_items`.

**Migrations:** golang-migrate, sequential numbering. Latest known is
**000019**; new work starts at **000020**.

---

## 2. What already exists for nesting (single-tier / depth-2)

### Schema delta from migration 000019

```sql
ALTER TABLE checklist_items
    ADD COLUMN type      TEXT NOT NULL DEFAULT 'task',
    ADD COLUMN parent_id UUID REFERENCES checklist_items(id) ON DELETE CASCADE;

ALTER TABLE checklist_items
    ADD CONSTRAINT chk_checklist_type CHECK (type IN ('task', 'note')),
    ADD CONSTRAINT chk_note_not_done  CHECK (type = 'task' OR done = false);

CREATE INDEX idx_checklist_plan_parent ON checklist_items (plan_id, parent_id);

CREATE OR REPLACE FUNCTION enforce_checklist_two_tier()
RETURNS TRIGGER AS $$
DECLARE parent_parent UUID; has_children BOOLEAN;
BEGIN
  IF NEW.parent_id IS NOT NULL THEN
    SELECT parent_id INTO parent_parent FROM checklist_items WHERE id = NEW.parent_id;
    IF parent_parent IS NOT NULL THEN
      RAISE EXCEPTION 'two-tier nesting max: cannot nest under a child item (id=%)', NEW.parent_id;
    END IF;
  END IF;
  IF TG_OP = 'UPDATE' AND NEW.parent_id IS NOT NULL AND (OLD.parent_id IS NULL OR OLD.parent_id <> NEW.parent_id) THEN
    SELECT EXISTS(SELECT 1 FROM checklist_items WHERE parent_id = NEW.id) INTO has_children;
    IF has_children THEN
      RAISE EXCEPTION 'two-tier nesting max: row % has children and cannot become a child', NEW.id;
    END IF;
  END IF;
  RETURN NEW;
END; $$ LANGUAGE plpgsql;

CREATE TRIGGER trg_checklist_two_tier
    BEFORE INSERT OR UPDATE ON checklist_items
    FOR EACH ROW EXECUTE FUNCTION enforce_checklist_two_tier();
```

### Full `checklist_items` columns (current)

```
checklist_items
├── id           UUID PK
├── plan_id      UUID FK → plans ON DELETE CASCADE
├── parent_id    UUID FK → self  ON DELETE CASCADE, nullable
├── description  TEXT NOT NULL
├── done         BOOLEAN NOT NULL DEFAULT false
├── sequence     INTEGER NOT NULL
├── scope        TEXT NOT NULL DEFAULT 'longterm'  CHECK IN ('daily','longterm')
├── type         TEXT NOT NULL DEFAULT 'task'      CHECK IN ('task','note')
├── start_date   DATE, nullable     -- Plan Calendar Gantt
├── due_date     DATE, nullable     -- CHECK start_date <= due_date when both set
├── archived     BOOLEAN DEFAULT false
├── created_at   TIMESTAMPTZ
└── updated_at   TIMESTAMPTZ
```

### Service-layer guards (`internal/checklistitems/service.go`)

- `validateParentID(ctx, planID, parentID)` — parent must exist, belong
  to same plan, and itself be top-level (`parent_id IS NULL`).
- `Create`: runs `validateParentID` when `parent_id` provided.
- `Update`: same validation; also rejects re-parenting a row that already
  has children (would push them past tier 2); rejects `task → note`
  flip on a row with children; auto-clears `done` on `task → note`
  transition when current row is `done = true`.

### API endpoints already in place

| Method | Path | Notes |
|---|---|---|
| GET    | `/api/plans/:id/checklists?scope=&type=` | Both filters optional |
| POST   | `/api/plans/:id/checklists` | Body: `description`, optional `scope`, `type`, `parentId` |
| PATCH  | `/api/plans/:id/checklists/:checklist_id` | Body may set `type`; `parentId` is 3-state via a custom `optUUID` JSON unmarshaller (absent / null / uuid) |
| DELETE | `/api/plans/:id/checklists/:checklist_id` | Cascades to children via FK |

### Frontend nesting UX (context only — don't modify)

- `Tab` / `Shift+Tab` on a focused or editing row → indent / outdent
- Indent walks upward through render order past child rows to find the
  nearest top-level parent
- Children render with `ml-8` + thin left guide

---

## 3. What we're building

Rebuild item nesting and add a sibling concept of task dependencies:

1. **Arbitrary-depth hierarchy** — drop the 2-tier cap. Goals →
   Milestones → Tasks → Subtasks → …
2. **Task dependencies** — a task can block one or more other tasks.
   The dependency graph is a DAG (cycles forbidden).
3. **Roll-up state** — a parent's "% complete" derived from its
   descendant leaf tasks; "earliest blocker due" derived from
   descendants and their dependencies.

This is intentionally a bigger redesign than the per-feature slice
work that produced the current schema. The point is to feel multiple
DB-design trade-offs in one feature.

---

## 4. Design exercises (the learning core)

For each: the mentor presents the options, asks which I'd pick given
expected access patterns, then walks me through the chosen
implementation. Sketch the alternatives in code comments so I can
explain when I'd switch later.

### 4.1 Hierarchical storage

Current code uses an **adjacency list** (`parent_id` self-FK).
Compare and pick:

- **Adjacency list + recursive CTE** — keep the column, query with
  `WITH RECURSIVE`. Reads expensive at depth, writes trivial.
- **Materialized path** — add `path TEXT` like `/uuid1/uuid2/uuid3/`.
  Reads cheap with `LIKE` prefix, inserts trivial, moves rewrite all
  descendants.
- **`ltree` extension** — same idea, native operators, GiST index.
- **Closure table** — separate `(ancestor_id, descendant_id, depth)`
  storing every ancestor-descendant pair. Reads cheapest; writes
  proportional to depth; moves require O(N×M) updates.

For Fireplace's expected scale (≤ ~100 items per plan, depth typically
≤ 4) a closure table is overkill, **but the point is to learn it.**
Build it. Maintain via triggers.

### 4.2 Roll-up state

A goal at the top of the tree shows "X% complete" from descendants'
`done` field. Walk me through:

- **Compute on read** with recursive CTE — always fresh, no write
  amplification, but every list page re-walks.
- **Materialized view + scheduled refresh** — fast reads, staleness
  window, refresh cost grows with data.
- **Trigger-maintained counter columns** on the parent — fast reads,
  always fresh, but every leaf update fires triggers up the tree
  (longer transactions, bigger deadlock surface).
- **Generated columns** — `GENERATED ALWAYS AS (...)` — limited
  because Postgres generated columns can only reference the same row.

Build one. Comment the alternatives.

### 4.3 Move subtree

Implement "move item X (with all its descendants) under item Y." This
is the **test case** for the hierarchical scheme:

- Adjacency list: trivially `UPDATE … SET parent_id = Y WHERE id = X`.
  But invariants like "Y can't be a descendant of X" must be checked
  first.
- Closure table: delete all rows in closure where
  `descendant IN X-subtree AND ancestor NOT IN X-subtree`, then insert
  new rows for the new ancestor chain. **Single transaction.**
- Materialized path / ltree: string rewrite of all descendants' paths.

Implement cycle prevention at both the service layer AND as a
`BEFORE UPDATE` trigger.

### 4.4 Task dependencies (DAG)

Add:

```
task_dependencies
├── blocker_id       UUID FK checklist_items ON DELETE CASCADE
├── blocked_id       UUID FK checklist_items ON DELETE CASCADE
├── dependency_type  TEXT  -- small enum, e.g. 'start_after_finish'
├── created_at       TIMESTAMPTZ
└── PRIMARY KEY (blocker_id, blocked_id)
```

Walk me through:

- **Same-plan-only** enforcement (deferrable constraint? trigger?
  service-layer? trade-offs?)
- **Cycle prevention on insert** via recursive CTE in a `BEFORE` trigger
- **Transitive blocked-by query** — "all tasks that depend on X being
  complete"
- **Topological sort** — "what's ready to start now"
- **Critical path** — longest chain of incomplete tasks reachable from
  a goal's leaves

### 4.5 Index design

After the schema settles, design the index set together:

- Partial indexes on `(plan_id, …) WHERE archived = false`
- Closure table: `(ancestor_id, depth)` plus PK
- GIN on `path` if `ltree` is chosen
- `task_dependencies (blocked_id)` for reverse lookups

---

## 5. Implementation plan (vertical slices)

Each slice is a working state: commit-worthy, tests pass.

### Slice 1 — Closure table foundation

- Migration 000020: create `checklist_closure (ancestor_id,
  descendant_id, depth, PRIMARY KEY (ancestor_id, descendant_id))`
- Drop `enforce_checklist_two_tier` trigger and its function (or guard
  with a removal migration); keep `parent_id` column
- Trigger functions `maintain_closure_after_insert/delete` /
  `maintain_closure_on_parent_change` to keep closure in sync
- Self-row in closure for each item (`ancestor = descendant`,
  `depth = 0`) — simplifies queries
- Backfill closure for existing rows in the same migration
- Smoke tests via `psql` inside `BEGIN ... ROLLBACK`: insert chain of
  5, query ancestors with `depth <= 3`, delete subtree

### Slice 2 — Service + API for arbitrary depth

- Remove depth-2 service guards (the validate-parent-is-top-level
  check)
- New repo methods:
  - `GetSubtree(rootID) -> []ChecklistItem`
  - `GetAncestors(itemID) -> []ChecklistItem`
  - `MoveSubtree(itemID, newParentID)` — transactional
- Service: move-subtree with cycle prevention (target not in subtree)
- Decide: extend existing PATCH (set `parentId` to anything in the
  same plan) or add `PATCH .../move`
- Tests: cycle rejection, move with descendants, depth assertions

### Slice 3 — Roll-up completion %

- Pick ONE strategy (suggested: trigger-maintained counters for the
  learning value)
- Add `descendant_count`, `descendant_done_count` columns
- Triggers on INSERT/UPDATE/DELETE of `checklist_items` that walk
  closure up to update ancestor counters
- Expose `completion_pct` in API responses
- Tests: insert leaf → ancestors' counters update; toggle leaf done
  → counters update; move subtree → source ancestors decrement, target
  ancestors increment

### Slice 4 — Dependencies table + cycle prevention

- Migration: `task_dependencies` table
- Trigger function for cycle detection (recursive CTE)
- Same-plan enforcement (trigger or service-layer; discuss)
- Service: `AddDependency`, `RemoveDependency`, `ListDependencies`
- API: `POST /api/plans/:plan_id/dependencies`,
  `DELETE /api/plans/:plan_id/dependencies/:blocker_id/:blocked_id`,
  `GET /api/plans/:plan_id/checklists/:id/dependencies?direction=blocks|blocked-by`
- Tests: cycle → trigger rejects; cross-plan → service rejects;
  chain of 5 → toposort correct

### Slice 5 — Derived queries

- `GET /api/plans/:id/ready-tasks` — `done = false` AND every blocker
  is `done = true`
- `GET /api/plans/:id/checklists/:id/critical-path` — longest chain of
  incomplete tasks from this item via dependencies
- Implement each as a single recursive CTE; sketch a Go-side
  equivalent for comparison
- Tests: hand-built graphs with known answers

### Slice 6 — Frontend (optional for the DB track)

- Replace the `ml-8`-flat render with a recursive tree component
- `Tab` handler sets `parent_id` to the previous sibling at any depth
  (drop the "walk to top-level" loop)
- Dependencies as inline badges; click filters the list to the blocker

This is meaningful UI work; safe to skip and stay on the DB side.

---

## 6. Testing approach

- Use the dev DB (Docker compose, port 5500, database `fireplace_gateway`) — or the mirrored test instance on 6500 (ADR-0010)
- Heavy use of `psql` for trigger/constraint smoke tests inside
  `BEGIN ... ROLLBACK` blocks before writing Go tests
- Service tests follow the existing `mockRepository` pattern in
  `internal/checklistitems/service_test.go`
- Repo-level Go tests aren't standard in this codebase — `make test`
  covers service-and-up
- The pre-existing `internal/user/service_test.go` build failure is
  unrelated; expect it as a known-broken baseline

---

## 7. Conventions to follow

- Commits: `feat(scope): description (#issue-number)`. Bug fixes use
  `fix(scope):`. No issue number is fine for this learning track.
- Migrations: `000NNN_descriptive_name.{up,down}.sql`. Always write a
  working down migration; smoke-test the round-trip on the dev DB.
- New columns: include in `RETURNING` clauses of `INSERT` queries; add
  to every `SELECT` column list in the repo.
- When changing `models.ChecklistItem`, update
  `internal/models/entities_test.go` to lock the new JSON shape.
- Errors at service layer wrap one of `constants.ErrInvalidInput`,
  `ErrNotFound`, `ErrForbidden`. Handlers `errors.Is`-check to map to
  400/404/403.

---

## 8. Out of scope for this exercise

- Multi-user collaborative editing on dependencies
- Migrating existing 2-tier data to deeper trees — it just keeps its
  current depth
- Performance benchmarking under load (do it later)
- Replacing `parent_id` — keep it as the canonical "direct parent"
  pointer; closure table augments rather than replaces it
- UI for dependency creation (mentor can sketch but skip implementing)
