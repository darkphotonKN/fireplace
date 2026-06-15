-- Self-contained migration for the insights-service DB (ordinal 6).
-- insights-service owns no plan/checklist data at runtime — it reads that from
-- plan-service over gRPC and generates suggestions via an LLM. This starter
-- table backs a future "cache generated insights" feature so the reserved DB is
-- wired and migrations run cleanly from day one. Carries its own trigger
-- function (no cross-service table dependencies after the strangler split).

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS generated_insights (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL,
    user_id UUID NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('suggestion', 'daily', 'video')),
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fast "latest insights for a plan of a given kind" lookups.
CREATE INDEX idx_generated_insights_plan_kind ON generated_insights (plan_id, kind, created_at DESC);

CREATE TRIGGER update_generated_insights_updated_at
    BEFORE UPDATE ON generated_insights
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
