-- Migration: 000006_add_focus_to_plans.down.sql
ALTER TABLE plans
DROP COLUMN focus;
