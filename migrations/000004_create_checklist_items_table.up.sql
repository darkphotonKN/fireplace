CREATE TABLE IF NOT EXISTS checklist_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID REFERENCES plans(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    done BOOLEAN NOT NULL DEFAULT false,
    sequence INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for faster sorting by sequence
CREATE INDEX idx_checklist_items_sequence ON checklist_items(sequence);

-- Index for filtering by project
CREATE INDEX idx_checklist_items_plan ON checklist_items(plan_id);

-- Trigger to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_checklist_items_modtime
BEFORE UPDATE ON checklist_items
FOR EACH ROW
EXECUTE FUNCTION update_modified_column();
