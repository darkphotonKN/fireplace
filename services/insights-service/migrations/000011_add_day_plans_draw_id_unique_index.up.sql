-- Every tick and untick finds its entry's plan by joining day_plan_entries to
-- day_plans on draw_id (FindByEntry), and a draw belongs to exactly one plan:
-- a new draw id is minted per draw. Index it, unique, so the join is an index
-- lookup and a draw id can never name two plans.
CREATE UNIQUE INDEX day_plans_draw_id_key ON day_plans (draw_id);
