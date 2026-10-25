-- The day plan (FS-02JBZ R1–R5, R11, S1): up to six entries in up to three
-- labelled groups per (plan, user, plan date), stored beside the daily brief.
--
-- day_plans: one row per key, created by the key's first draw and updated in
-- place by every later one. draw_id names the draw the row currently serves:
-- a redraw replaces the plan by moving draw_id to a new draw, conditionally on
-- the draw it started from, so of two racing redraws exactly one is stored and
-- the other reads it (R21). free_draw_used records that a successful AI draw
-- was stored for the key: until then a draw is free, after it every AI draw
-- spends from refresh_counters, the counter the brief's refresh also spends
-- from (R15, A3). A rules plan or an empty no-candidates plan leaves it false
-- (R16, R18).
CREATE TABLE day_plans (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id         UUID NOT NULL,
    user_id         UUID NOT NULL,
    plan_date       DATE NOT NULL,
    draw_id         UUID NOT NULL,
    source          TEXT NOT NULL CHECK (source IN ('ai', 'rules')),
    generated_at    TIMESTAMPTZ NOT NULL,
    plan_state_hash TEXT NOT NULL,
    free_draw_used  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT day_plans_key UNIQUE (plan_id, user_id, plan_date)
);

-- update_updated_at_column() is defined by migration 000001.
CREATE TRIGGER update_day_plans_updated_at
    BEFORE UPDATE ON day_plans
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- day_plan_entries: append-only, one row per entry of one draw, in drawn order
-- (position across the whole plan) with its group's order and label. A group is
-- a label over entries, with no id of its own, so it is not a table. Entries of
-- a replaced draw stay stored under their draw_id: a reader that read the old
-- draw_id still reads a whole plan (R11), and entry ids stay valid as suggestion
-- ids (FS-02JBZ O1). Plan items are referenced by id only: plan-service owns
-- them, so there is no foreign key.
CREATE TABLE day_plan_entries (
    id             UUID PRIMARY KEY,
    day_plan_id    UUID NOT NULL REFERENCES day_plans (id) ON DELETE CASCADE,
    draw_id        UUID NOT NULL,
    position       SMALLINT NOT NULL CHECK (position BETWEEN 0 AND 5),
    group_position SMALLINT NOT NULL CHECK (group_position BETWEEN 0 AND 2),
    group_label    TEXT NOT NULL CHECK (btrim(group_label) = group_label AND char_length(group_label) BETWEEN 1 AND 40),
    kind           TEXT NOT NULL CHECK (kind IN ('existing_item', 'new_step')),
    item_id        UUID,
    parent_item_id UUID,
    text           TEXT NOT NULL CHECK (btrim(text) <> ''),
    reason         TEXT NOT NULL CHECK (btrim(reason) <> ''),
    minutes        SMALLINT CHECK (minutes BETWEEN 5 AND 240),
    revision       SMALLINT CHECK (revision BETWEEN 1 AND 3),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- An entry names an item iff it points at an existing one; a new step
    -- always names the parent it would go under.
    CONSTRAINT day_plan_entries_item_iff_existing CHECK ((kind = 'existing_item') = (item_id IS NOT NULL)),
    CONSTRAINT day_plan_entries_new_step_has_parent CHECK (kind <> 'new_step' OR parent_item_id IS NOT NULL),
    CONSTRAINT day_plan_entries_draw_position_key UNIQUE (draw_id, position)
);

-- Usage of the day plan's generation is recorded as day_plan (R6).
ALTER TABLE generations DROP CONSTRAINT generations_kind_check;
ALTER TABLE generations ADD CONSTRAINT generations_kind_check CHECK (kind IN (
    'daily_brief', 'step_suggestions', 'item_tree',
    'checklist_suggestion', 'video_terms', 'daily_suggestions',
    'day_plan'
));
