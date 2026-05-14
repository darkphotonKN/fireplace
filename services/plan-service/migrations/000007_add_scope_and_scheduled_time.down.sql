-- Migration: 000007_add_scope_and_scheduled_time.down.sql
DROP INDEX IF EXISTS idx_checklist_items_scope;
ALTER TABLE checklist_items 
DROP COLUMN scheduled_time,
DROP COLUMN scope;
