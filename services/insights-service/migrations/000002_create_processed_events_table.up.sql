-- Append-only dedup / idempotency ledger for the insights-service AMQP consumer.
-- One row is inserted the first time an (event_id, consumer) pair is handled; the
-- consumer checks existence before processing so redelivered events are no-ops.
-- Rows are never updated, so there is deliberately no updated_at / update trigger.

CREATE TABLE IF NOT EXISTS processed_events (
    event_id UUID NOT NULL,
    consumer TEXT NOT NULL DEFAULT 'insights', -- always insights
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, consumer) -- enforces the dedup
);

-- Supports time-based cleanup/retention scans of old processed rows.
CREATE INDEX idx_processed_events ON processed_events(created_at);
