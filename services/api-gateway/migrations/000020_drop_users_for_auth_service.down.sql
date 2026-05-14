-- Rollback for Phase 3 cutover. Recreates the users table with the schema
-- the gateway last had (post-000014: name, email, password, display_name,
-- bio). FK constraints are NOT re-added automatically — the dependent
-- columns may now contain UUIDs that don't exist in this database (since
-- the source of truth moved to auth-service). Add them manually after a
-- data backfill if rolling all the way back.

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    display_name TEXT,
    bio TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
