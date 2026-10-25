-- Day ticks (FS-02JBZ R24-R30, D1): an entry's item marked done for the day
-- plan's date, per (user, plan, item, date), with the entry it was ticked
-- through. A tick is a fact about one user's day, not about the item, so it
-- lives here and never touches plan-service's done (R28); on a shared plan
-- (FS-12ZMF) every member has their own. Plan items are referenced by id only:
-- plan-service owns them, so there is no foreign key. Untick deletes the row
-- (R29).
CREATE TABLE day_ticks (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL,
    plan_id    UUID NOT NULL,
    item_id    UUID NOT NULL,
    plan_date  DATE NOT NULL,
    entry_id   UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT day_ticks_key UNIQUE (user_id, plan_id, plan_date, item_id)
);

-- Ticked entries survive a redraw with their ids (R12, R39): the new draw
-- stores its own row for each, under the same entry id. An entry id is
-- therefore unique per draw, not across the table, and lookups by id (tick,
-- untick) go through the index below.
--
-- Binding a new step on its first tick (R25) updates that entry's row in the
-- draw the key serves, in place: the one exception to append-only, and only
-- ever new_step -> existing_item.
ALTER TABLE day_plan_entries DROP CONSTRAINT day_plan_entries_pkey;
ALTER TABLE day_plan_entries ADD CONSTRAINT day_plan_entries_pkey PRIMARY KEY (draw_id, id);
CREATE INDEX day_plan_entries_id_idx ON day_plan_entries (id);
