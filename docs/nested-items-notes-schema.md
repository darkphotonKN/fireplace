# Nested Items + Notes — Schema

Adds a `type` discriminator and a self-referencing `parent_id` to `checklist_items` to support inline notes and a single level of nesting (two-tier max).

## 1. Schema Changes

### checklist_items (modified)

| Column     | Type | Notes                                                                                          |
| ---------- | ---- | ---------------------------------------------------------------------------------------------- |
| type       | TEXT | NOT NULL, DEFAULT `'task'`. CHECK `IN ('task', 'note')`.                                       |
| parent_id  | UUID | nullable. FK → `checklist_items(id) ON DELETE CASCADE`. Two-tier max enforced by trigger.      |

### Constraints (added)

| Name                       | Definition                                                                                          |
| -------------------------- | --------------------------------------------------------------------------------------------------- |
| `chk_checklist_type`       | `CHECK (type IN ('task', 'note'))`                                                                  |
| `chk_note_not_done`        | `CHECK (type = 'task' OR done = false)` — notes can never be done                                   |
| `trg_checklist_two_tier`   | Trigger BEFORE INSERT/UPDATE: if `NEW.parent_id IS NOT NULL` and the referenced row's `parent_id IS NOT NULL`, raise exception. Also fires on UPDATE if a row with existing children gets `parent_id` set (would push children past tier 2). |

### Indexes (added)

| Name                              | Columns                | Purpose                                                            |
| --------------------------------- | ---------------------- | ------------------------------------------------------------------ |
| `idx_checklist_plan_parent`       | `(plan_id, parent_id)` | Fast "children of X" and "top-level items in plan" queries         |

## 2. Migration Sketch

```sql
-- up
ALTER TABLE checklist_items
  ADD COLUMN type      TEXT    NOT NULL DEFAULT 'task',
  ADD COLUMN parent_id UUID    REFERENCES checklist_items(id) ON DELETE CASCADE;

ALTER TABLE checklist_items
  ADD CONSTRAINT chk_checklist_type CHECK (type IN ('task', 'note')),
  ADD CONSTRAINT chk_note_not_done  CHECK (type = 'task' OR done = false);

CREATE INDEX idx_checklist_plan_parent ON checklist_items (plan_id, parent_id);

CREATE OR REPLACE FUNCTION enforce_checklist_two_tier()
RETURNS TRIGGER AS $$
DECLARE
  parent_parent UUID;
  has_children  BOOLEAN;
BEGIN
  IF NEW.parent_id IS NOT NULL THEN
    SELECT parent_id INTO parent_parent FROM checklist_items WHERE id = NEW.parent_id;
    IF parent_parent IS NOT NULL THEN
      RAISE EXCEPTION 'two-tier nesting max: cannot nest under a child item (id=%)', NEW.parent_id
        USING ERRCODE = '23514';
    END IF;
  END IF;

  -- A row with existing children cannot itself become a child (would push children past tier 2)
  IF TG_OP = 'UPDATE' AND NEW.parent_id IS NOT NULL AND OLD.parent_id IS NULL THEN
    SELECT EXISTS(SELECT 1 FROM checklist_items WHERE parent_id = NEW.id) INTO has_children;
    IF has_children THEN
      RAISE EXCEPTION 'two-tier nesting max: row % has children and cannot become a child', NEW.id
        USING ERRCODE = '23514';
    END IF;
  END IF;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_checklist_two_tier
  BEFORE INSERT OR UPDATE ON checklist_items
  FOR EACH ROW
  EXECUTE FUNCTION enforce_checklist_two_tier();
```

```sql
-- down
DROP TRIGGER IF EXISTS trg_checklist_two_tier ON checklist_items;
DROP FUNCTION IF EXISTS enforce_checklist_two_tier();

DROP INDEX IF EXISTS idx_checklist_plan_parent;

ALTER TABLE checklist_items DROP CONSTRAINT IF EXISTS chk_note_not_done;
ALTER TABLE checklist_items DROP CONSTRAINT IF EXISTS chk_checklist_type;

ALTER TABLE checklist_items DROP COLUMN parent_id;
ALTER TABLE checklist_items DROP COLUMN type;
```

## 3. Service-Layer Guards (BE)

Even with the trigger as a backstop, the service layer should pre-validate to return clean HTTP 400s instead of generic DB errors:

- `parent_id` (when set) must reference a row in the same `plan_id`
- `parent_id` cannot reference a row whose `parent_id IS NOT NULL`
- A row that has children cannot itself be re-parented
- `scope='daily'` items cannot be created via `POST /checklists` (must come from the AI suggestion accept flow)
- `type='note'` flips `done` to `false` automatically when transitioning from `task` with `done=true`

## 4. Out of Scope

- Drag-to-nest UX (Tab / Shift+Tab and the indent icon are the only nesting affordances)
- More than two tiers
- UX for parent delete (warning dialogs, undo) — DB cascades unconditionally for now
- Restoring the dedicated Note block on the right side of the plan layout
