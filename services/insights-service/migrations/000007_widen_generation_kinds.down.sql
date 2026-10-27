DROP INDEX IF EXISTS generations_created_at_idx;

-- The narrower check cannot hold rows of the kinds it no longer allows, so
-- reverting drops them from the ledger.
DELETE FROM generations WHERE kind IN ('checklist_suggestion', 'video_terms', 'daily_suggestions');
ALTER TABLE generations DROP CONSTRAINT generations_kind_check;
ALTER TABLE generations ADD CONSTRAINT generations_kind_check CHECK (kind IN (
    'daily_brief', 'step_suggestions', 'item_tree'
));
