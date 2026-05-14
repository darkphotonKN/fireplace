-- Migration: 000008_update_scope_values.up.sql
-- First remove the existing constraint
ALTER TABLE checklist_items
DROP CONSTRAINT check_valid_scope;

-- Update existing values
UPDATE checklist_items
SET scope = 'longterm'
WHERE scope IN ('project', 'global');

-- Add new constraint with updated values
ALTER TABLE checklist_items
ADD CONSTRAINT check_valid_scope CHECK (scope IN ('longterm', 'daily'));
