-- The refresh counter (FS-DF4ZB R17, L5): counted refreshes per (plan, user,
-- local date). Keyed by the day, not by the brief, so the day plan's redraws
-- (FS-02JBZ) can share the same cap without a second counter.
--
-- One row per key, created by the first counted refresh. The cap is enforced
-- by the single conditional upsert that takes a refresh (spent < cap), so the
-- check and the increment are one atomic step and concurrent refreshes cannot
-- overspend it. A key with no row has spent nothing.
CREATE TABLE refresh_counters (
    plan_id    UUID NOT NULL,
    user_id    UUID NOT NULL,
    plan_date  DATE NOT NULL,
    spent      SMALLINT NOT NULL CHECK (spent >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT refresh_counters_pkey PRIMARY KEY (plan_id, user_id, plan_date)
);

-- update_updated_at_column() is defined by migration 000001.
CREATE TRIGGER update_refresh_counters_updated_at
    BEFORE UPDATE ON refresh_counters
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
