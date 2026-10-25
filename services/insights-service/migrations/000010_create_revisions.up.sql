-- The revision ladder (FS-02JBZ R41-R48, D5, D6, S6, S7): on learning plans,
-- an item's first day tick ever schedules revision 1, and ticking revision N
-- completes it and schedules N+1, chained from that tick's plan date (+1, +3,
-- +7 days), at most 3. Per user: on a shared plan (FS-12ZMF) every member has
-- their own ladder. Plan items are referenced by id only: plan-service owns
-- them, so there is no foreign key, and a revision of an item since archived
-- or deleted is filtered out against the plan tree when read (R44).
--
-- scheduled_on and completed_on are the plan dates of the ticks that
-- scheduled and completed the rung: an untick on a date undoes exactly what
-- that date's tick did (R46, D5), reopening the rung completed on it and
-- removing the rung scheduled on it.
CREATE TABLE revisions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL,
    plan_id      UUID NOT NULL,
    item_id      UUID NOT NULL,
    rung         SMALLINT NOT NULL CHECK (rung BETWEEN 1 AND 3),
    scheduled_on DATE NOT NULL,
    due_on       DATE NOT NULL CHECK (due_on > scheduled_on),
    completed_on DATE CHECK (completed_on IS NULL OR completed_on >= due_on),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT revisions_key UNIQUE (user_id, plan_id, item_id, rung)
);

-- Due revisions for a plan date are the open rungs due on or before it.
CREATE INDEX revisions_open_idx ON revisions (user_id, plan_id, due_on) WHERE completed_on IS NULL;

-- The per-item revisions switch (R47, R48), on by default: an item with no
-- row is on. Switching off writes enabled = false and cancels the open rung;
-- switching back on flips an existing row to true, which tells the item's
-- next day tick to resume the ladder at the rung after the last completed.
-- Switching on an item that was never off writes nothing.
CREATE TABLE revision_switches (
    user_id    UUID NOT NULL,
    plan_id    UUID NOT NULL,
    item_id    UUID NOT NULL,
    enabled    BOOLEAN NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id, plan_id, item_id)
);

-- "First day tick ever" (R42) asks whether the item has a tick on any other
-- date; day_ticks_key leads with the date, so it cannot answer that.
CREATE INDEX day_ticks_item_idx ON day_ticks (user_id, plan_id, item_id);
