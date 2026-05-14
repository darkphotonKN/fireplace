-- Down migration. Best-effort: scheduled_time time-of-day cannot be recovered
-- from a DATE column; restored values are midnight UTC.

CREATE TABLE IF NOT EXISTS calendar_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    checklist_item_id UUID REFERENCES checklist_items(id) ON DELETE SET NULL,
    entry_type TEXT NOT NULL CHECK (entry_type IN ('daily', 'longterm', 'recommendation')),
    scheduled_date DATE NOT NULL,
    position INTEGER NOT NULL CHECK (position >= 1 AND position <= 8),
    pinned BOOLEAN NOT NULL DEFAULT false,
    rec_title TEXT,
    rec_url TEXT,
    rec_description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_calendar_plan_date_pos ON calendar_entries (plan_id, scheduled_date, position);
CREATE INDEX IF NOT EXISTS idx_calendar_plan_date ON calendar_entries (plan_id, scheduled_date);
CREATE INDEX IF NOT EXISTS idx_calendar_checklist_item ON calendar_entries (checklist_item_id);

CREATE TRIGGER update_calendar_entries_updated_at
    BEFORE UPDATE ON calendar_entries
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

DROP INDEX IF EXISTS idx_checklist_plan_due_date;
DROP INDEX IF EXISTS idx_checklist_plan_start_date;

ALTER TABLE checklist_items ADD COLUMN scheduled_time TIMESTAMPTZ;

UPDATE checklist_items
SET scheduled_time = start_date::timestamptz
WHERE start_date IS NOT NULL;

ALTER TABLE checklist_items DROP CONSTRAINT IF EXISTS chk_checklist_dates;
ALTER TABLE checklist_items DROP COLUMN due_date;
ALTER TABLE checklist_items DROP COLUMN start_date;
