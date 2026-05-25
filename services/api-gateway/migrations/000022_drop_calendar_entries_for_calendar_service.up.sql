-- Cutover migration: calendar_entries now lives in calendar-service's own DB.
-- The gateway never actually queried this table (calendar's repository read
-- checklist_items, not calendar_entries), so dropping it from the gateway DB
-- is purely a cleanup.
DROP TRIGGER IF EXISTS update_calendar_entries_updated_at ON calendar_entries;
DROP TABLE IF EXISTS calendar_entries;
