-- Migration: replace scheduled_time with start_date + due_date on checklist_items.
-- Plan Calendar (Gantt) — see PRD #34 / issue #35.

ALTER TABLE checklist_items
    ADD COLUMN start_date DATE,
    ADD COLUMN due_date   DATE;

UPDATE checklist_items
SET start_date = scheduled_time::date
WHERE scheduled_time IS NOT NULL;

ALTER TABLE checklist_items
    ADD CONSTRAINT chk_checklist_dates
    CHECK (start_date IS NULL OR due_date IS NULL OR start_date <= due_date);

ALTER TABLE checklist_items DROP COLUMN scheduled_time;

CREATE INDEX idx_checklist_plan_start_date ON checklist_items (plan_id, start_date);
CREATE INDEX idx_checklist_plan_due_date   ON checklist_items (plan_id, due_date);

-- calendar_entries table is superseded by start_date / due_date on checklist_items.
DROP TRIGGER IF EXISTS update_calendar_entries_updated_at ON calendar_entries;
DROP TABLE IF EXISTS calendar_entries;
