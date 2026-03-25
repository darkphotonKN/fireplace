-- Create trigger function if it doesn't exist
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create notes table
CREATE TABLE IF NOT EXISTS notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'user',
    priority VARCHAR(20) NOT NULL DEFAULT 'medium',
    tags TEXT[], -- Array of tags
    related_task_ids UUID[], -- Array of related checklist item IDs
    is_read BOOLEAN DEFAULT false,
    is_dismissed BOOLEAN DEFAULT false,
    ai_metadata JSONB, -- Store AI generation metadata as JSON
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better query performance
CREATE INDEX idx_notes_plan_id ON notes(plan_id);
CREATE INDEX idx_notes_type ON notes(type);
CREATE INDEX idx_notes_priority ON notes(priority);
CREATE INDEX idx_notes_is_read ON notes(is_read);
CREATE INDEX idx_notes_is_dismissed ON notes(is_dismissed);
CREATE INDEX idx_notes_created_at ON notes(created_at DESC);

-- Create GIN index for array columns
CREATE INDEX idx_notes_tags ON notes USING GIN(tags);
CREATE INDEX idx_notes_related_task_ids ON notes USING GIN(related_task_ids);

-- Add trigger to update updated_at timestamp
CREATE TRIGGER update_notes_updated_at
    BEFORE UPDATE ON notes
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();