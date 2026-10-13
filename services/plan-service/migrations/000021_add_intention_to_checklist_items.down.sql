ALTER TABLE checklist_items DROP CONSTRAINT IF EXISTS chk_checklist_intention_length;

ALTER TABLE checklist_items DROP COLUMN intention;
