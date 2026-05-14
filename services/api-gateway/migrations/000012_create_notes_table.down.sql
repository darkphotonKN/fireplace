-- Drop notes table and all its indexes
DROP TRIGGER IF EXISTS update_notes_updated_at ON notes;
DROP TABLE IF EXISTS notes;