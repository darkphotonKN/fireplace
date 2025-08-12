DROP TRIGGER IF EXISTS update_user_analytics_updated_at ON user_analytics;
DROP INDEX IF EXISTS idx_user_analytics_user_date;
DROP TABLE IF EXISTS user_analytics;