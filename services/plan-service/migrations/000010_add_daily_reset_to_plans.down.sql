-- Migration: 000010_add_daily_reset_to_plans.down.sql

-- Remove the index first
DROP INDEX IF EXISTS idx_plans_daily_reset;

-- Then remove the column
ALTER TABLE plans
DROP COLUMN IF EXISTS daily_reset;
