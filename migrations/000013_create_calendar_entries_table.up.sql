-- Create calendar_entries table
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

-- One entry per slot per day per plan
CREATE UNIQUE INDEX uq_calendar_plan_date_pos ON calendar_entries (plan_id, scheduled_date, position);

-- Fast month-range lookups
CREATE INDEX idx_calendar_plan_date ON calendar_entries (plan_id, scheduled_date);

-- Reactive updates on task changes
CREATE INDEX idx_calendar_checklist_item ON calendar_entries (checklist_item_id);

-- Add trigger to update updated_at timestamp
CREATE TRIGGER update_calendar_entries_updated_at
    BEFORE UPDATE ON calendar_entries
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
