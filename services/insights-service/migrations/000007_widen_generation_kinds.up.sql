-- The usage ledger becomes load-bearing (FS-02JBZ R31, A4; ADR-0019): every
-- model call is recorded, including the single checklist suggestion, the video
-- search terms and the legacy daily suggestions, which were unrecorded.
ALTER TABLE generations DROP CONSTRAINT generations_kind_check;
ALTER TABLE generations ADD CONSTRAINT generations_kind_check CHECK (kind IN (
    'daily_brief', 'step_suggestions', 'item_tree',
    'checklist_suggestion', 'video_terms', 'daily_suggestions'
));

-- The token budget sums today's (UTC) rows before every model call (R32).
CREATE INDEX generations_created_at_idx ON generations (created_at);
