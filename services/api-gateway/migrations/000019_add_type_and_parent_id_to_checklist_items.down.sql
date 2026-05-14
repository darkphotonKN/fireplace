DROP TRIGGER IF EXISTS trg_checklist_two_tier ON checklist_items;
DROP FUNCTION IF EXISTS enforce_checklist_two_tier();

DROP INDEX IF EXISTS idx_checklist_plan_parent;

ALTER TABLE checklist_items DROP CONSTRAINT IF EXISTS chk_note_not_done;
ALTER TABLE checklist_items DROP CONSTRAINT IF EXISTS chk_checklist_type;

ALTER TABLE checklist_items DROP COLUMN parent_id;
ALTER TABLE checklist_items DROP COLUMN type;
