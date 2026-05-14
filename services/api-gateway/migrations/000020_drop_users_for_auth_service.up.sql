-- Phase 3 cutover: users + authentication now live in auth-service.
-- The api-gateway no longer owns the users table.
--
-- This migration drops the FK constraints from plan-scoped tables (plans,
-- user_analytics, plan_shares) and then drops the users table itself.
-- user_id columns remain on those tables as plain UUIDs; cross-service
-- integrity is enforced at the application/event layer from now on.

ALTER TABLE IF EXISTS plans            DROP CONSTRAINT IF EXISTS plans_user_id_fkey;
ALTER TABLE IF EXISTS user_analytics   DROP CONSTRAINT IF EXISTS user_analytics_user_id_fkey;
ALTER TABLE IF EXISTS plan_shares      DROP CONSTRAINT IF EXISTS plan_shares_user_id_fkey;

DROP TABLE IF EXISTS users;
