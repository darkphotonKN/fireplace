DROP INDEX IF EXISTS day_plan_entries_id_idx;
-- Entry ids repeated across draws by pinning must go before the old key returns.
DELETE FROM day_plan_entries e
USING day_plan_entries newer
WHERE e.id = newer.id AND e.created_at < newer.created_at;
ALTER TABLE day_plan_entries DROP CONSTRAINT day_plan_entries_pkey;
ALTER TABLE day_plan_entries ADD CONSTRAINT day_plan_entries_pkey PRIMARY KEY (id);

DROP TABLE IF EXISTS day_ticks;
