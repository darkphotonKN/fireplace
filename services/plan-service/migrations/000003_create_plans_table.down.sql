DROP TRIGGER IF EXISTS update_plans_modtime ON plans;
-- Don't drop the function in case it's used by other tables
DROP INDEX IF EXISTS idx_plans_user;
DROP TABLE IF EXISTS plans;
