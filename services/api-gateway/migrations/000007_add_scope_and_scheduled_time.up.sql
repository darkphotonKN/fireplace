-- Migration: 000007_add_scope_and_scheduled_time.up.sql
ALTER TABLE checklist_items
ADD COLUMN scope TEXT NOT NULL DEFAULT 'project',
ADD COLUMN scheduled_time TIMESTAMP WITH TIME ZONE;

-- Add check constraint to validate scope values
ALTER TABLE checklist_items
ADD CONSTRAINT check_valid_scope CHECK (scope IN ('global', 'daily', 'project'));

-- Create index for faster filtering by scope
CREATE INDEX idx_checklist_items_scope ON checklist_items(scope);
