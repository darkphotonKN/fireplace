-- Migration: 000006_add_focus_to_plans.up.sql
ALTER TABLE plans
ADD COLUMN focus TEXT;

UPDATE plans
SET focus = 'temp default focus: please update';

ALTER TABLE plans
ALTER COLUMN focus SET NOT NULL;
