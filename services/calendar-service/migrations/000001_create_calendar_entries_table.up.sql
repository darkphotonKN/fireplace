-- Self-contained migration for the calendar-service DB.
-- The original (in api-gateway/migrations/000013) assumed plans + checklist_items
-- + update_updated_at_column were already in the DB. After the strangler split
-- those references are gone; this migration carries its own trigger function
-- and drops the cross-table FKs.

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create calendar_entries table
CREATE TABLE IF NOT EXISTS calendar_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL,
    checklist_item_id UUID,
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

-- One entry per slot per day per plan
CREATE UNIQUE INDEX uq_calendar_plan_date_pos ON calendar_entries (plan_id, scheduled_date, position);

-- Fast month-range lookups
CREATE INDEX idx_calendar_plan_date ON calendar_entries (plan_id, scheduled_date);

-- Reactive updates on task changes (still useful as an app-level index even though
-- the FK is gone — plan-service owns checklist_items now).
CREATE INDEX idx_calendar_checklist_item ON calendar_entries (checklist_item_id);

-- Add trigger to update updated_at timestamp
CREATE TRIGGER update_calendar_entries_updated_at
    BEFORE UPDATE ON calendar_entries
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
