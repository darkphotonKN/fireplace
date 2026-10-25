-- Drops the table only. The vector extension is superuser-owned and was never
-- this migration's to create, so it is not this migration's to drop.
DROP TABLE IF EXISTS rag_chunks;
