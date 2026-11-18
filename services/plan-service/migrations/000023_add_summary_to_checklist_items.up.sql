-- Migration: add summary to checklist_items.
-- FS-PGG2N R2/R3/R9: a short summary a note carries, set through the item update.
-- plan-service trims it and stores empty as NULL. Only a note may carry one:
-- chk_summary_note_only is the twin of chk_note_not_done.

ALTER TABLE checklist_items
    ADD COLUMN summary TEXT;

ALTER TABLE checklist_items
    ADD CONSTRAINT chk_checklist_summary_length
        CHECK (summary IS NULL OR char_length(summary) <= 280);

ALTER TABLE checklist_items
    ADD CONSTRAINT chk_summary_note_only
        CHECK (type = 'note' OR summary IS NULL);
