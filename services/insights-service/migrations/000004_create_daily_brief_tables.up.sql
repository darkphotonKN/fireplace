-- Guided suggestions: daily briefs, their suggestions, and generation usage
-- (FS-DF4ZB R9, R14, R15, R22).
--
-- daily_briefs: at most one brief per (plan, user, local date). The unique key
-- is what makes concurrent first reads store one brief (R22): the losing insert
-- does nothing and reads the winner.
CREATE TABLE daily_briefs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id      UUID NOT NULL,
    user_id      UUID NOT NULL,
    brief_date   DATE NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT daily_briefs_key UNIQUE (plan_id, user_id, brief_date)
);

-- update_updated_at_column() is defined by migration 000001.
CREATE TRIGGER update_daily_briefs_updated_at
    BEFORE UPDATE ON daily_briefs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- suggestions: one row per proposal shown to a user, with the id feedback will
-- reference. Today every row is a move of a daily brief; step and item-tree
-- suggestions (R25) will widen kind and leave brief_id NULL. Plan items are
-- referenced by id only: plan-service owns them, so there is no foreign key.
CREATE TABLE suggestions (
    id             UUID PRIMARY KEY,
    user_id        UUID NOT NULL,
    plan_id        UUID NOT NULL,
    brief_id       UUID REFERENCES daily_briefs (id) ON DELETE CASCADE,
    position       SMALLINT NOT NULL CHECK (position >= 0),
    kind           TEXT NOT NULL CHECK (kind IN ('existing_step', 'new_step')),
    parent_item_id UUID NOT NULL,
    step_item_id   UUID,
    text           TEXT NOT NULL CHECK (btrim(text) <> ''),
    reason         TEXT NOT NULL CHECK (btrim(reason) <> ''),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- A move names a step iff it points at an existing one.
    CONSTRAINT suggestions_step_iff_existing CHECK ((kind = 'existing_step') = (step_item_id IS NOT NULL)),
    CONSTRAINT suggestions_brief_position_key UNIQUE (brief_id, position)
);

-- generations: the usage ledger. One row per model call, failed or not,
-- including a draw that lost the R22 race. Reading it is a query (no UI).
CREATE TABLE generations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind                TEXT NOT NULL CHECK (kind IN ('daily_brief', 'step_suggestions', 'item_tree')),
    plan_id             UUID NOT NULL,
    user_id             UUID NOT NULL,
    model               TEXT NOT NULL,
    input_tokens        INTEGER NOT NULL CHECK (input_tokens >= 0),
    cached_input_tokens INTEGER NOT NULL CHECK (cached_input_tokens >= 0),
    output_tokens       INTEGER NOT NULL CHECK (output_tokens >= 0),
    succeeded           BOOLEAN NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
