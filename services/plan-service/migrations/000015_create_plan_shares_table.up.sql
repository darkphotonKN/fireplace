CREATE TABLE IF NOT EXISTS plan_shares (
    user_id UUID NOT NULL,
    plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'edit',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id, plan_id)
);

-- Prevent owner from sharing with themselves (handled in app logic too, but belt-and-suspenders)
ALTER TABLE plan_shares
ADD CONSTRAINT check_valid_role CHECK (role IN ('edit', 'view'));

-- Fast lookup: "which plans are shared with me?"
CREATE INDEX idx_plan_shares_user ON plan_shares(user_id);

-- Fast lookup: "who has access to this plan?"
CREATE INDEX idx_plan_shares_plan ON plan_shares(plan_id);

CREATE TRIGGER update_plan_shares_modtime
BEFORE UPDATE ON plan_shares
FOR EACH ROW
EXECUTE FUNCTION update_modified_column();
