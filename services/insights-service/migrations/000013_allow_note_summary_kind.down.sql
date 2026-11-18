-- Rows recorded as note_summary would violate the narrowed check, so they go
-- first.
DELETE FROM generations WHERE kind = 'note_summary';
ALTER TABLE generations DROP CONSTRAINT generations_kind_check;
ALTER TABLE generations ADD CONSTRAINT generations_kind_check CHECK (kind IN (
    'daily_brief', 'step_suggestions', 'item_tree',
    'checklist_suggestion', 'video_terms', 'daily_suggestions',
    'day_plan'
));
