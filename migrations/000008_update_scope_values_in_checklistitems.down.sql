-- Migration: 000008_update_scope_values.down.sql
-- Restore original constraint
ALTER TABLE checklist_items
DROP CONSTRAINT check_valid_scope;

-- Restore original values - set everything back to 'project' as default
UPDATE checklist_items
SET scope = 'project'
WHERE scope = 'longterm';

-- Restore original constraint
ALTER TABLE checklist_items
ADD CONSTRAINT check_valid_scope CHECK (scope IN ('global', 'daily', 'project'));
