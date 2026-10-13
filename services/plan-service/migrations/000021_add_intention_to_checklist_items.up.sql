-- Migration: add intention to checklist_items.
-- FS-043WP R37/R40: free text an item carries, set through the item update.
-- The server gives it no meaning; plan-service trims it and stores empty as NULL.

ALTER TABLE checklist_items
    ADD COLUMN intention TEXT;

ALTER TABLE checklist_items
    ADD CONSTRAINT chk_checklist_intention_length
        CHECK (intention IS NULL OR char_length(intention) <= 280);
