-- Rollback for Phase 4c cutover. Recreates plans + checklist_items + plan_shares
-- + resources with the schema the gateway last had. FK constraints are NOT
-- re-added automatically — see 000020 down for the same caveat.

CREATE TABLE IF NOT EXISTS plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID,
    name TEXT NOT NULL,
    description TEXT,
    focus TEXT,
    plan_type TEXT NOT NULL,
    daily_reset BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS checklist_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL,
    description TEXT NOT NULL,
    done BOOLEAN NOT NULL DEFAULT FALSE,
    sequence INTEGER NOT NULL DEFAULT 0,
    scope TEXT NOT NULL DEFAULT 'longterm',
    type TEXT NOT NULL DEFAULT 'task',
    parent_id UUID,
    start_date DATE,
    due_date DATE,
    archived BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS plan_shares (
    user_id UUID NOT NULL,
    plan_id UUID NOT NULL,
    role TEXT NOT NULL DEFAULT 'edit',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, plan_id)
);

CREATE TABLE IF NOT EXISTS resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id UUID NOT NULL,
    url TEXT NOT NULL,
    title TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
