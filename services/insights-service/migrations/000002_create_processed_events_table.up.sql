
CREATE TABLE IF NOT EXISTS processed_events (
    event_id UUID NOT NULL,
    consumer TEXT NOT NULL DEFAULT 'insights', -- always insights
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, consumer) -- enforces the dedup
);

CREATE INDEX idx_processed_events ON processed_events(created_at)
