-- Phase 4c cutover: plans + checklist_items now live in plan-service.
-- Drop the FK constraints from notes and calendar_entries (they still hold
-- a plain plan_id UUID, but the column is no longer enforced — integrity is
-- handled application-side via AssertPlanOwnership gRPC calls before write).
-- Then drop the now-unused plan-side tables.

ALTER TABLE IF EXISTS notes              DROP CONSTRAINT IF EXISTS notes_plan_id_fkey;
ALTER TABLE IF EXISTS calendar_entries   DROP CONSTRAINT IF EXISTS calendar_entries_plan_id_fkey;

DROP TABLE IF EXISTS plan_shares;
DROP TABLE IF EXISTS resources;
DROP TABLE IF EXISTS checklist_items;
DROP TABLE IF EXISTS plans;
