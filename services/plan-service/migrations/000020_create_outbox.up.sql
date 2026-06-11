CREATE TABLE outbox (
    -- also used as event_id
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Core routing and event metadata
    routing_key VARCHAR(255) NOT NULL,
    exchange VARCHAR(255) NOT NULL,
    payload BYTEA NOT NULL,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- State of published
    -- null = pending 
    -- not null (existing timestamp) = published
    -- whole row not present = no issues
    published_at TIMESTAMPTZ NULL
);

CREATE INDEX idx_outbox_pending ON outbox(created_at) WHERE published_at IS NULL;

