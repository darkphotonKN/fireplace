-- Rollback for the calendar_entries drop. Recreates the table without the
-- cross-service FK constraints (plans + checklist_items live in other DBs).
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
