DROP TRIGGER IF EXISTS update_resources_modtime ON resources;
DROP INDEX IF EXISTS idx_resources_sequence;
DROP INDEX IF EXISTS idx_resources_user;
DROP INDEX IF EXISTS idx_resources_plan;
DROP TABLE IF EXISTS resources;
