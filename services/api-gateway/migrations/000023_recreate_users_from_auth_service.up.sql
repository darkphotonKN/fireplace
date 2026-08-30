-- ADR-0009 §1: auth-service folded back into the api-gateway, so the gateway
-- owns the users table again.
--
-- History: the gateway created users in 000001 and dropped it in 000020 when
-- auth-service was extracted. This is not a revert of that migration — it is a
-- forward recreation matching auth-service's schema as it stood at the fold
-- (its 000002 + the profile fields from its 000003), which had diverged from
-- the gateway's original 000001.
--
-- The FK constraints 000020 dropped (plans, user_analytics, plan_shares) are
-- NOT restored: plans live in plan-service's database now, so those columns
-- stay plain UUIDs with integrity enforced at the application/event layer.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

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
