# plan-service — Specification

<!-- migrating to thin format: one line per capability, → FS-NNNN pointers -->

> Scope: plans, checklist items, Gantt dates, nesting/notes, sharing, resources. Platform maps: ../../CLAUDE.md.

plan-service owns the core domain of Fireplace: Plans and Checklist Items. This spec
covers only what this service owns and serves. Cross-domain concerns (calendar
read-model, HTTP REST surface, AI suggestions) are listed under **Owned elsewhere**.

---

## Domain Terms

| Term             | Definition                                                                                   |
| ---------------- | -------------------------------------------------------------------------------------------- |
| Plan             | A project or learning goal container — `plan_type`: `project` or `learning`                  |
| Checklist Item   | A task or note within a plan, scoped `daily` (recurring) or `longterm` (milestone)           |
| Scope            | Whether an item is `daily` (short-term, repeatable) or `longterm` (milestone-based)          |
| Focus            | A plan's stated objective (`focus`, NOT NULL) — fed to AI by insights-service for context    |
| Daily Reset      | Bulk reset of completed daily-scoped items to `done=false` (this service's `DailyReset` RPC) |
| Task             | Checklist item with `type='task'` (default) — supports the `done` checkbox                   |
| Note             | Checklist item with `type='note'` — text-only; no checkbox, `done` always false              |
| Parent Item      | A top-level item (`parent_id IS NULL`) that may have children                                |
| Child Item       | A nested item whose `parent_id` references its parent                                        |
| Two-Tier Nesting | Parents may have children, children may not — depth capped at exactly two tiers              |
| Gantt range      | An item's `[start_date, due_date]` (both nullable DATE) — its position on the Plan Calendar  |
| Plan Share       | A `plan_shares` row granting another user `edit` or `view` access to a plan                  |

---

## Features (BE, this service)

### Implemented

**Plans**

- [x] CRUD for plans (create / read / update / delete)
- [x] Plan types `project` | `learning`; `focus` field (NOT NULL)
- [x] `daily_reset` toggle per plan (`ToggleDailyReset` RPC) + default-by-plan-type on create
      (`project` → false, else true)
- [x] Plan name search (`SearchPlans`, user-scoped `ILIKE`, trigram GIN index)
- [x] Plan sharing via `plan_shares`; `ListSharedPlans` (owned ∪ shared-with-me)
- [x] `AssertPlanOwnership` — cheap ownership check for sibling services
- [x] Cascade-delete a user's plans on `user.deleted` (consumer)
- [x] `plan.created` published via transactional outbox

**Checklist Items**

- [x] CRUD within a plan; scope `daily` | `longterm`; auto-incrementing `sequence`
- [x] Archive / unarchive (`ArchiveItem`)
- [x] Bulk daily reset (`DailyReset` RPC; respects plan `daily_reset`)
- [x] `ListItemsByUser` (non-archived, across a user's plans — for useranalytics)
- [x] `checklist_item.completed` / `.uncompleted` events on done-flip (direct publish)

**Plan Calendar (data side)**

- [x] `start_date` / `due_date` DATE columns (replaced `scheduled_time`; `calendar_entries` dropped)
- [x] `UpdateItemDates` — set/clear either date; validates `start_date <= due_date`
- [x] `ListItemsInDateWindow` — items whose `[start,due]` range intersects a window
- [x] `ListUpcomingItems` — items starting within the next week

**Nested Items + Notes**

- [x] `type` (`task`|`note`) + `parent_id` self-FK on `checklist_items`
- [x] Notes can never be `done` (DB CHECK + service guard)
- [x] Two-tier nesting max (DB trigger + service `validateParent`)
- [x] Indent/outdent via `UpdateItem` (`parent_id` set / cleared); `?type=` filter on `ListItems`

### In Progress / Partial

- [ ] `plan.deleted` event — marshaled but publish is **commented out** (stub); nothing emitted yet
- [ ] Daily-items-only-via-AI rule — spec calls for rejecting manual `scope='daily'` creation;
      **not currently enforced** in `checklistitem.service.Create` (validation allows `daily`)

### Future (not started, this service's slice)

- [ ] AI auto-scheduling of item dates (would consume from insights-service)
- [ ] Pinning / locking items to dates
- [ ] Serving the `resources` table (schema-only today)

**Security / Identity**

- [ ] gRPC caller identity from context — a shared JWT interceptor validates the caller
      (RS256, verify-only) and injects the user identity into `context.Context`; handlers
      read identity from context and cross-check the legacy `req.user_id` body field
      (mismatch → `Unauthenticated`). Phase 1 of the platform-wide auth distribution;
      constraint in ADR-0001. → FS-0001

---

## Data Model

### plans

| Column      | Type          | Notes                                            |
| ----------- | ------------- | ------------------------------------------------ |
| id          | UUID PK       | `gen_random_uuid()`                              |
| user_id     | UUID          | owner (no in-DB FK — users live in auth-service) |
| name        | TEXT NOT NULL |                                                  |
| focus       | TEXT NOT NULL | added 000006                                     |
| description | TEXT          |                                                  |
| plan_type   | TEXT NOT NULL | `project` \| `learning` (no DB CHECK; app-level) |
| daily_reset | BOOL NOT NULL | DEFAULT true (000010)                            |
| created_at  | TIMESTAMPTZ   | DEFAULT NOW()                                    |
| updated_at  | TIMESTAMPTZ   | trigger-updated                                  |

Indexes: `idx_plans_user(user_id)`, `idx_plans_daily_reset(daily_reset)`,
`idx_plans_name_trgm` GIN `gin_trgm_ops` on `name` (000017; `pg_trgm`).
Trigger: `update_plans_modtime` bumps `updated_at`.

### checklist_items

| Column      | Type             | Notes                                                         |
| ----------- | ---------------- | ------------------------------------------------------------- |
| id          | UUID PK          | `gen_random_uuid()`                                           |
| plan_id     | UUID             | FK → plans(id) **ON DELETE CASCADE**                          |
| description | TEXT NOT NULL    |                                                               |
| done        | BOOL NOT NULL    | DEFAULT false                                                 |
| sequence    | INTEGER NOT NULL |                                                               |
| scope       | TEXT NOT NULL    | DEFAULT `'project'`→ rewritten to `'longterm'` (see CHECK)    |
| archived    | BOOL             | DEFAULT false (000009)                                        |
| start_date  | DATE             | nullable — Gantt range start (000018)                         |
| due_date    | DATE             | nullable — Gantt range end (000018)                           |
| type        | TEXT NOT NULL    | DEFAULT `'task'` (000019)                                     |
| parent_id   | UUID             | nullable, self-FK → checklist_items(id) **ON DELETE CASCADE** |
| created_at  | TIMESTAMPTZ      | DEFAULT NOW()                                                 |
| updated_at  | TIMESTAMPTZ      | trigger-updated                                               |

> **`scheduled_time` (old scheduling column) was dropped in 000018** and replaced by
> `start_date`/`due_date`. The `calendar_entries` table was dropped in the same migration.

**CHECK constraints:**

- `check_valid_scope`: `scope IN ('longterm', 'daily')` — **the real values** (000008; see
  Business Rules). Default column value remains `'longterm'` in app code (repo `Create`).
- `chk_checklist_dates`: `start_date IS NULL OR due_date IS NULL OR start_date <= due_date`
- `chk_checklist_type`: `type IN ('task', 'note')`
- `chk_note_not_done`: `type = 'task' OR done = false` — notes can never be done

**Trigger** `trg_checklist_two_tier` (BEFORE INSERT/UPDATE): rejects a `parent_id` whose
referenced row is itself nested (tier-3 attempt) and rejects a row with children becoming a
child. Raises SQLSTATE `23514`.

**Indexes:** `idx_checklist_items_sequence`, `idx_checklist_items_plan`,
`idx_checklist_items_scope`, `idx_checklist_plan_start_date(plan_id, start_date)`,
`idx_checklist_plan_due_date(plan_id, due_date)`, `idx_checklist_plan_parent(plan_id, parent_id)`.

### plan_shares

| Column     | Type          | Notes                                        |
| ---------- | ------------- | -------------------------------------------- |
| user_id    | UUID          | PK part; grantee                             |
| plan_id    | UUID          | PK part; FK → plans(id) CASCADE              |
| role       | TEXT NOT NULL | DEFAULT `'edit'`; CHECK `IN ('edit','view')` |
| created_at | TIMESTAMPTZ   |                                              |
| updated_at | TIMESTAMPTZ   | trigger-updated                              |

PK `(user_id, plan_id)` (one share per user/plan; used for `INSERT ... ON CONFLICT DO
NOTHING` idempotency). Indexes: `idx_plan_shares_user`, `idx_plan_shares_plan`.

### resources — schema-only / NOT served

Table + indexes exist (migration 000005) but **no Go code references it**. Columns:
`id`, `plan_id` (FK → plans CASCADE), `resource_type`, `url`, `title`, `description`,
`sequence`, `created_at`, `updated_at`. Indexes `idx_resources_plan`, `idx_resources_sequence`.
Not exposed via any gRPC method.

### outbox

| Column       | Type                  | Notes                     |
| ------------ | --------------------- | ------------------------- |
| id           | UUID PK               | doubles as event_id       |
| routing_key  | VARCHAR(255) NOT NULL |                           |
| exchange     | VARCHAR(255) NOT NULL |                           |
| payload      | BYTEA NOT NULL        | marshaled protobuf event  |
| created_at   | TIMESTAMPTZ NOT NULL  | DEFAULT CURRENT_TIMESTAMP |
| published_at | TIMESTAMPTZ NULL      | NULL = pending            |

Partial index `idx_outbox_pending(created_at) WHERE published_at IS NULL`. Drained
`FOR UPDATE SKIP LOCKED`, batch limit 10, by the shared publish-worker (~every 2 min).

---

## gRPC Surface

### PlanService (`common/api/proto/plan`, :7103)

| Method              | Input                                              | Output              | Notes                                                   |
| ------------------- | -------------------------------------------------- | ------------------- | ------------------------------------------------------- |
| CreatePlan          | user_id, name, focus, desc, plan_type, daily_reset | Plan                | Writes plan + `plan.created` outbox row in one tx       |
| GetPlan             | id                                                 | Plan                |                                                         |
| ListPlans           | user_id                                            | ListPlansResponse   | Owner's plans, newest first                             |
| ListSharedPlans     | user_id, limit, offset                             | ListPlansResponse   | Owned ∪ shared-with-me (UNION over `plan_shares`)       |
| SearchPlans         | user_id, query, limit, offset                      | SearchPlansResponse | User-scoped `name ILIKE %query%`; returns id/name/desc  |
| UpdatePlan          | id, user*id, \_optional fields*                    | Plan                | COALESCE partial update; scoped by user_id              |
| ToggleDailyReset    | id, user_id                                        | Plan                | Flips `daily_reset`                                     |
| DeletePlan          | id, user_id                                        | Empty               | Deletes plan (children cascade); `plan.deleted` is stub |
| AssertPlanOwnership | plan_id, user_id                                   | Empty               | `ErrNotFound` if absent, `ErrForbidden` if other owner  |

### ChecklistService

| Method                | Input                                              | Output                           | Notes                                                    |
| --------------------- | -------------------------------------------------- | -------------------------------- | -------------------------------------------------------- |
| CreateItem            | plan_id, description, scope?, type?, parent_id?    | ChecklistItem                    | Validates scope/type/parent; sequence = count+1          |
| GetItem               | id                                                 | ChecklistItem                    |                                                          |
| ListItems             | plan_id, scope?, type?                             | ListItemsResponse                | Non-archived; ordered by sequence                        |
| ListArchivedItems     | plan_id                                            | ListItemsResponse                | `archived = true`                                        |
| ListUpcomingItems     | plan_id                                            | ListItemsResponse                | `start_date` within next week                            |
| ListItemsByUser       | user_id                                            | ListItemsResponse                | Non-archived across user's plans (useranalytics)         |
| ListItemsInDateWindow | plan_id, window_start, window_end                  | ListItemsResponse                | Range-intersection; excludes archived (calendar-service) |
| UpdateItem            | id, _optional fields_, parent_id / clear_parent_id | ChecklistItem                    | done-flip emits completed/uncompleted; parent 3-state    |
| UpdateItemDates       | id, start_date?, due_date?                         | ChecklistItem                    | Set/leave each; validates `start <= due`                 |
| ArchiveItem           | id, archived                                       | ChecklistItem                    |                                                          |
| DeleteItem            | id                                                 | Empty                            | Children cascade via FK                                  |
| DailyReset            | (none)                                             | DailyResetResponse (reset_count) | Bulk CTE reset; invoked by api-gateway cron              |

---

## Events

| Event                        | Dir | Exchange / Queue                            | Transport  | Notes                                                |
| ---------------------------- | --- | ------------------------------------------- | ---------- | ---------------------------------------------------- |
| `plan.created`               | pub | `plan.events`                               | **outbox** | Written in create-plan tx; drained by publish-worker |
| `checklist_item.completed`   | pub | `plan.events`                               | direct     | On `UpdateItem` setting `done=true`                  |
| `checklist_item.uncompleted` | pub | `plan.events`                               | direct     | On `UpdateItem` setting `done=false`                 |
| `plan.deleted`               | pub | `plan.events`                               | **stub**   | Marshaled but publish commented out — not emitted    |
| `user.deleted`               | sub | `auth.events` → queue `plan-service.events` | AMQP       | → `CascadeDeleteForUser`                             |

Consumer error policy: `ErrTransient` → `Nack(requeue=true)`; parse/permanent failures →
`Ack` (drop) to avoid a poison-message loop. Cascade delete is idempotent at the
"row already gone" boundary.

---

## Business Rules

**Daily Reset** — `DailyReset` RPC runs a single bulk CTE: `UPDATE checklist_items SET
done=false` where the item is `done=true`, `scope='daily'`, and its plan has
`daily_reset=true`. Row-by-row updates are not used. The nightly schedule (14:00 UTC cron)
that calls this RPC lives in **api-gateway**, not here.

**Checklist Scope Validation** — real allowed values are **`'daily'` and `'longterm'`**,
enforced by DB CHECK `check_valid_scope` (migration 000008) and by service-layer checks in
`Create` / `Update` / `ListByPlanID`. Default is `'longterm'`. (Note: the master product
spec lists the same `daily`|`longterm` pair; a mid-exploration reading of the migrations
suggested `global`|`daily`|`project`, but that was the _original_ 000007 constraint which
000008 replaced — the live constraint is `('longterm','daily')`.)

**Plan Calendar Date Validation** — `start_date` and `due_date` are independent nullable
DATE columns. When both are set, `start_date <= due_date` is enforced at the API (400 /
`ErrInvalidInput`) in `UpdateDates` — including the case where one date is in the request
and the other already exists in the DB — and at the DB via `chk_checklist_dates`. Clearing
both removes the item from calendar queries.

**Nested Items + Notes** —

- `type` defaults to `'task'`; `'note'` rows have `done` permanently `false` (DB CHECK).
- Switching a `task` with `done=true` to `note` force-sets `done=false` server-side.
- A `task` with children cannot be converted to a `note` (notes can't be parents).
- Two-tier max: a child cannot become a parent; a `parent_id` cannot reference an already
  nested row. Enforced by `validateParent` (service, `ErrInvalidInput`) and the DB trigger.
- A `parent_id` cannot reference a row in a different plan (service guard).

**Daily-items-only-via-AI (spec, not yet enforced)** — the product spec calls for
rejecting manual creation of `scope='daily'` items (they should come from insights-service's
accept flow). The current `Create` validation permits `daily`; this rule is **not yet
implemented** in plan-service.

**Sharing / Ownership** — `AssertPlanOwnership` returns `ErrNotFound` (absent) or
`ErrForbidden` (owned by another user). `plan_shares` grants `edit`/`view`; `ListSharedPlans`
returns owned plans UNION plans shared with the user. Share insert is idempotent
(`ON CONFLICT (user_id, plan_id) DO NOTHING`).

---

## Edge Cases

| Scenario                                        | Handling                                                                |
| ----------------------------------------------- | ----------------------------------------------------------------------- |
| PATCH dates with `start_date > due_date`        | 400 (`ErrInvalidInput`); neither persisted                              |
| Set only one date, other already violates order | 400 — validated against the current DB value                            |
| Item archived                                   | Excluded from `ListItemsInDateWindow` / `ListItems` regardless of dates |
| Both dates null                                 | Item excluded from date-window queries                                  |
| Indent under a row that is itself a child       | 400 (service `validateParent`); DB trigger is backstop (SQLSTATE 23514) |
| Re-parent a row that has its own children       | 400 — would push children to tier 3                                     |
| Parent belongs to a different plan              | 400 — cross-plan nesting forbidden                                      |
| Toggle `task`(done=true) → `note`               | `done` forced to `false`, then `type=note`; one Update commits both     |
| Toggle `note` → `task`                          | Type flips; `done` stays `false` until set                              |
| Convert a parent (has children) → `note`        | 400 — notes can't be parents                                            |
| Delete a parent with children                   | Children cascade-delete via FK                                          |
| Archive a parent                                | Parent archived only; children not touched (archive does not cascade)   |
| `user.deleted` for a user with many plans       | Each plan deleted; children cascade; best-effort, first error returned  |
| Duplicate `user.deleted` delivery               | Idempotent (missing rows are a no-op)                                   |

---

## Owned elsewhere (cross-refs)

- **Calendar read-model / window rendering** (GET `/calendar`, single-day chip vs Gantt
  bar, week/month grid) → **calendar-service**. This service only supplies the raw
  date-windowed items via `ListItemsInDateWindow`.
- **HTTP REST surface + drag/drop UI** (all `/api/plans/...` routes, `PATCH /dates`,
  indent/outdent keyboard flows, the nightly daily-reset cron) → **api-gateway** and
  **client** (flow-client).
- **AI daily suggestions** that feed `daily`-scoped items (checklist suggestion / daily
  suggestion accept flow) → **insights-service**.
- **User accounts / `user.deleted` emission** → **auth-service**.
- **User analytics** consuming `ListItemsByUser` → **useranalytics**.
