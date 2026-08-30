-- FIX (I-0039): this migration used update_updated_at_column() but the function
-- was only DEFINED in 000012, one migration later. The lineage was therefore
-- never runnable from an empty database — it only ever succeeded against the
-- original monolith DB, which already carried the function. Nothing surfaced it
-- until migrations began running against a fresh database under ADR-0010.
--
-- Defining it here is safe and idempotent: CREATE OR REPLACE is a no-op on a
-- database that already has it, and any existing database is already past
-- version 11, so this file never re-runs there. 000012 keeps its own identical
-- CREATE OR REPLACE and is unaffected.
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE user_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    
    -- Simple daily counts
    tasks_completed INTEGER NOT NULL DEFAULT 0,
    tasks_total INTEGER NOT NULL DEFAULT 0,
    completion_rate DECIMAL(3,2) DEFAULT 0, -- 0.00 to 1.00
    
    -- Basic streak (just consecutive days with >0 completed tasks)
    current_streak INTEGER DEFAULT 0,
    
    -- Simple time tracking
    active_plans_count INTEGER DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- one record per day
    UNIQUE(user_id, date)
);

CREATE INDEX idx_user_analytics_user_date ON user_analytics(user_id, date DESC);

-- Create trigger for updated_at
CREATE TRIGGER update_user_analytics_updated_at BEFORE UPDATE ON user_analytics
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
