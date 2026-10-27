-- Migration: retire the daily scope, down (FS-02JBZ R50).
--
-- Deliberately a NO-OP. The up migration is lossy: once a daily row is
-- long-term, nothing records that it was ever daily, and its old sequence is
-- gone too. Guessing (for example, flipping rows back by some heuristic) would
-- turn genuine long-term items daily, which is worse than leaving the data as
-- it is. Rolling back past 000022 therefore leaves every item long-term; the
-- schema is unchanged by the up migration, so there is nothing else to undo.
SELECT 1;
