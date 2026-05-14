-- Migration: 000012_add_trigram_index_to_plans.down.sql

DROP INDEX IF EXISTS idx_plans_name_trgm;

-- Don't drop pg_trgm extension in case other tables use it
