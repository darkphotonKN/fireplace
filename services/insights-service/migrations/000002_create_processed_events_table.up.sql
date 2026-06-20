
CREATE TABLE IF NOT EXISTS processed_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id UUID NOT NULL,
    consumer TEXT NOT NULL DEFAULT "insights", -- always insights
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
);

CREATE INDEX idx_processed_events ON processed_events(created_at)
