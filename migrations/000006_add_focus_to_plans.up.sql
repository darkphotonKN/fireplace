-- Migration: 000006_add_focus_to_plans.up.sql
ALTER TABLE plans
ADD COLUMN focus TEXT NOT NULL;
