ALTER TABLE checklist_items DROP CONSTRAINT IF EXISTS chk_summary_note_only;

ALTER TABLE checklist_items DROP CONSTRAINT IF EXISTS chk_checklist_summary_length;

ALTER TABLE checklist_items DROP COLUMN summary;
