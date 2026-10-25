-- rag_chunks: retrieval storage for guided suggestions (FS-DF4ZB R31–R33, ADR-0018).
--
-- ASSUMES the vector extension already exists in this database. It is never
-- created here: pgvector is untrusted, so only the instance superuser can
-- create it, and migrations run as fireplace_insights_owner (ADR-0010 §2).
-- A fresh volume gets it from scripts/db/init-databases.sh; an existing one
-- needs the one-off in deploy/README.md. Without it this fails with
-- `type "vector" does not exist`.
--
-- One row per embedded passage. user_id NULL marks a shared (playbook) row
-- readable by everyone; a personal (completion) row carries its user and the
-- plan its item lives in, so retrieval can exclude the plan being suggested for.
-- Every row records the model and dimension count that produced its vector, so
-- a model change is a re-embed, never a silent mix.
--
-- No ANN index (HNSW/IVFFlat), on purpose (R33): queries are exact scans, which
-- always return the true top k. An approximate index filters after the walk and
-- can return fewer than k under a per-user filter.

CREATE TABLE rag_chunks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID,
    plan_id         UUID,
    source          TEXT NOT NULL CHECK (source IN ('playbook', 'completion')),
    source_id       TEXT NOT NULL,
    chunk_no        INTEGER NOT NULL CHECK (chunk_no >= 0),
    content         TEXT NOT NULL,
    content_hash    TEXT NOT NULL,
    embedding       vector(512) NOT NULL,
    embedding_model TEXT NOT NULL CHECK (embedding_model <> ''),
    embedding_dims  INTEGER NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- R32: one row per passage. A second write with the same key replaces it.
    CONSTRAINT rag_chunks_source_chunk_key UNIQUE (source, source_id, chunk_no),

    -- A playbook row is shared (no user, no plan); a completion row is personal
    -- and always knows its user and plan (R31, R26).
    CONSTRAINT rag_chunks_scope_matches_source CHECK (
        (source = 'playbook'   AND user_id IS NULL     AND plan_id IS NULL) OR
        (source = 'completion' AND user_id IS NOT NULL AND plan_id IS NOT NULL)
    ),

    -- The recorded dimension count is the vector's own, never a label.
    CONSTRAINT rag_chunks_dims_match_vector CHECK (embedding_dims = vector_dims(embedding))
);

-- Personal-row lookups: "does this user have any personal rows" (the rule gate)
-- and the tenant filter. A plain btree, not a vector index.
CREATE INDEX idx_rag_chunks_user_id ON rag_chunks (user_id) WHERE user_id IS NOT NULL;

-- update_updated_at_column() is defined by migration 000001.
CREATE TRIGGER update_rag_chunks_updated_at
    BEFORE UPDATE ON rag_chunks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
