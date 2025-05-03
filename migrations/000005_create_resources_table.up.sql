CREATE TABLE IF NOT EXISTS resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID REFERENCES plans(id) ON DELETE CASCADE,
    resource_type TEXT NOT NULL, -- Github, Youtube, Udemy, etc
    url TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    sequence INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for filtering by plan
CREATE INDEX idx_resources_plan ON resources(plan_id);
-- Index for ordering resources
CREATE INDEX idx_resources_sequence ON resources(sequence);

CREATE TRIGGER update_resources_modtime
BEFORE UPDATE ON resources
FOR EACH ROW
EXECUTE FUNCTION update_modified_column();
