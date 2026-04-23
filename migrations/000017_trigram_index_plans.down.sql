CREATE EXTENSION IF NOT EXISTS pg_trgm;
 
CREATE INDEX idx_plans_name_trgm ON plans USING gin (name gin_trgm_ops);
 
