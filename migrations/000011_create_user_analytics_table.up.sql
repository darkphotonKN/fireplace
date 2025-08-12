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
