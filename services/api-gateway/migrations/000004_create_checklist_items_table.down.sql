DROP TRIGGER IF EXISTS update_checklist_items_modtime ON checklist_items;
DROP FUNCTION IF EXISTS update_modified_column();
DROP INDEX IF EXISTS idx_checklist_items_project;
DROP INDEX IF EXISTS idx_checklist_items_sequence;
DROP TABLE IF EXISTS checklist_items;
