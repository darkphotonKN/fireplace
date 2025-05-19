-- Migration: 000010_add_daily_reset_to_plans.up.sql

-- Add daily_reset column to plans table with default value of true
ALTER TABLE plans
ADD COLUMN daily_reset BOOLEAN NOT NULL DEFAULT true;

-- Add an index to improve query performance when filtering by daily_reset
CREATE INDEX idx_plans_daily_reset ON plans(daily_reset);

-- Add comment to document the column's purpose
COMMENT ON COLUMN plans.daily_reset IS 'Controls whether daily items in this plan should be automatically reset each day';
