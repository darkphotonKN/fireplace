-- Rows recorded as day_plan would violate the narrowed check, so they go first.
DELETE FROM generations WHERE kind = 'day_plan';
ALTER TABLE generations DROP CONSTRAINT generations_kind_check;
ALTER TABLE generations ADD CONSTRAINT generations_kind_check CHECK (kind IN (
    'daily_brief', 'step_suggestions', 'item_tree',
    'checklist_suggestion', 'video_terms', 'daily_suggestions'
));

DROP TABLE IF EXISTS day_plan_entries;
DROP TABLE IF EXISTS day_plans;
