-- Placeholder initial migration for the auth-service.
-- Phase 3 will add the real users + refresh_tokens schema; for now we just
-- ensure the uuid extension is available so the migrator has something to run.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
