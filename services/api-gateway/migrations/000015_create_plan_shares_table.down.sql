DROP TRIGGER IF EXISTS update_plan_shares_modtime ON plan_shares;
DROP INDEX IF EXISTS idx_plan_shares_plan;
DROP INDEX IF EXISTS idx_plan_shares_user;
DROP TABLE IF EXISTS plan_shares;
