-- A note summary is a metered model call of its own kind (FS-PGG2N R41,
-- G15; ADR-0019). Constraint change only: no table is added.
ALTER TABLE generations DROP CONSTRAINT generations_kind_check;
ALTER TABLE generations ADD CONSTRAINT generations_kind_check CHECK (kind IN (
    'daily_brief', 'step_suggestions', 'item_tree',
    'checklist_suggestion', 'video_terms', 'daily_suggestions',
    'day_plan', 'note_summary'
));
