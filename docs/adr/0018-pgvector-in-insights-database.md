# ADR-0018 — Retrieval vectors live in fireplace_insights, on pgvector

Status: accepted
Date: 2026-10-13
Scope: root — governs the shared Postgres image, which database may hold the `vector`
extension, and the shape of retrieval storage in insights-service
Related: ADR-0009 (every model call belongs to insights-service), ADR-0010 (one instance, three
databases, DDL split out of the runtime role), ADR-0012 (the single VPS this has to fit on),
ADR-0014 (milestones stay a client reading)
Realized by: FS-DF4ZB (guided daily moves with retrieval)

## Context

Guided daily moves need retrieval: before insights-service asks a model what someone should do
today, it pulls the handful of passages most relevant to that person's plans and puts them in
the prompt. Retrieval needs embeddings stored somewhere, and a nearest-neighbour query over
them that can be narrowed to one user.

Nothing in the stack does this yet. The shared instance runs `postgres:17`, which has no vector
type. The extensions `init-databases.sh` installs today are `uuid-ossp` and `pg_trgm`.

Four places to put the vectors were weighed:

- **A dedicated vector database** (Pinecone, Qdrant). Built for the job, but Pinecone is a
  hosted dependency with its own bill and Qdrant is another container on the 1-2GB box of
  ADR-0012. Either way it is a second data store to back up, restore-drill and keep consistent
  with the first. At this corpus size that buys nothing Postgres can't do.
- **`real[]` columns, brute-force cosine in Go.** Works at this scale, and needs no new
  extension. But insights-service would load candidate rows and rank them itself, so tenant
  filtering, ordering and `LIMIT k` move out of SQL and into application code. That also gives
  up the standard operator story (`<=>`, index access methods) the moment the corpus outgrows a
  scan.
- **pgvector in another service's database.** Rejected outright. ADR-0009 §3 makes
  insights-service the home of every model call, and embeddings are model output. ADR-0010 §3
  keeps plan data and insights data in databases that cannot reach each other. Vectors in
  `fireplace_plans` or `fireplace_gateway` would break both.
- **pgvector in `fireplace_insights`.** Same instance, same backup path, same migration
  lineage as the rest of insights-service's data, and filtering stays a `WHERE` clause.

**pgvector is not a trusted extension, and that collides with ADR-0010.** Its `vector.control`
(pgvector 0.8.7, checked against the upstream repository) has no `trusted = true` line. The
PostgreSQL 17 `CREATE EXTENSION` docs say that installing an untrusted extension needs the
privileges to create everything inside it, which for a C extension means superuser. Only a
*trusted* extension can be installed by a non-superuser with `CREATE` on the database.
`<db>_owner` therefore **cannot** run `CREATE EXTENSION vector`, and `<db>_app` (no `CREATE`
at all, ADR-0010 §2) certainly can't. This differs from `uuid-ossp` and `pg_trgm`, which are
trusted. `init-databases.sh` creates those as the owner on purpose, so a `--clean` restore run
as the owner can drop and recreate them. That reason is written next to the `SET ROLE` in the
script, and it was found by running the restore drill.

Recorded without adversarial review: locked collaboratively with the owner and not run
through `challenge-me`.

## Decision

**Retrieval vectors are stored in `fireplace_insights` with pgvector. Nowhere else.**

1. **The shared instance image changes from `postgres:17` to `pgvector/pgvector:pg17`.** Same
   Postgres major, so the data directory carries over. The image just ships the extension's
   files. The test instance (ADR-0010 §5) changes with it, because the test tier mirrors
   production.
2. **The `vector` extension is created in `fireplace_insights` only.** `fireplace_plans` and
   `fireplace_gateway` never get it. The library being present in the image is not permission
   to use it elsewhere.
3. **A superuser bootstrap step creates the extension, not a migration.** Because pgvector is
   untrusted, `CREATE EXTENSION vector` runs as the instance superuser in the init step on the
   shared instance (`init-databases.sh` today), outside the `SET ROLE <owner>` block. Migrations
   run as `fireplace_insights_owner` and **assume the extension exists**. They never try to
   create it. The runtime role keeps no `CREATE` (ADR-0010 §2 stands unchanged). The superuser
   step installs the extension and nothing else.
4. **Retrieval data lives in one table, `rag_chunks`, in `fireplace_insights`.**
   - Embeddings are `vector(512)`, from `text-embedding-3-small` requested with
     `dimensions=512`.
   - **Every row stores its embedding model and dimension count.** A chunk's vector is
     meaningless without the model that produced it. Queries compare only rows from the current
     model. That makes re-embedding a migration that can run in the background (write
     new-model rows, cut queries over, delete old-model rows) instead of a stop-the-world
     rewrite.
   - **A `user_id` column scopes each row. `NULL` means a shared-corpus row** visible to
     everyone. Tenant scoping is a SQL filter (`user_id = $1 OR user_id IS NULL`), not a
     namespace per user and not a table per tenant.
   - Chunks taken from plan content describe plans and checklist items. "Milestone" stays a
     client reading (ADR-0014) and does not become a column, a chunk type or a server name.
5. **Exact scan, no ANN index, for now.** No HNSW or IVFFlat index until a single user's
   filtered scan stops being cheap. An exact scan always returns the true top k. An approximate
   index applies the `WHERE` filter *after* it walks the index, so a filtered query can return
   **fewer than k rows**. pgvector's docs say so plainly, and it hits hardest on a selective
   per-user filter.

## Consequences

**Good**

- No new service, container or data store. Retrieval data rides the instance, backups and
  migration step that ADR-0010 and ADR-0012 already pay for.
- Tenant filtering, ranking and `LIMIT k` stay in one SQL statement, so ranking stays out of
  application code.
- Isolation is unchanged. The vectors sit inside the database ADR-0010 already walls off. A
  plan-service query cannot reach them, and `fireplace_plans` can't even declare the type.
- Per-row model and dimension columns mean an embedding-model change doesn't strand the corpus.
- Exact scan means retrieval results are correct by construction until there is a measured
  reason to trade correctness for speed.

**Bad / accepted**

- **The extension is superuser-owned, an exception to the owner-creates-extensions rule in
  `init-databases.sh`.** It has to be: an untrusted extension can't be created any other way.
  But it brings back the exact restore failure that rule exists to prevent. Dumps are
  `--clean --if-exists` (`deploy/backup/backup.sh`). The restore runs as the owner with
  `ON_ERROR_STOP=1` (`deploy/backup/restore.sh`). A dump of `fireplace_insights` will carry
  `DROP EXTENSION IF EXISTS vector`, which fails for a non-owner, and on a fresh target its
  `CREATE EXTENSION` fails too. **The `fireplace_insights` restore drill breaks the day the
  extension lands unless backup or restore changes in the same slice.** The likely mechanism
  is `pg_dump --exclude-extension=vector` (available in pg_dump 17), relying on the target's
  own bootstrap to have created it. That is a mechanism choice for the implementing FS, not
  part of this decision. What is decided is that the drill must stay green.
- **The bootstrap step only runs on an empty volume.** Postgres runs
  `/docker-entrypoint-initdb.d` once, on first boot. Every existing volume needs a one-off
  `CREATE EXTENSION vector` in `fireplace_insights`, run by hand as the superuser: production,
  local dev, and the test instance unless it is recreated. That is a deploy step someone has
  to remember, and the migration that creates `rag_chunks` fails with `type "vector" does not
  exist` until it is done. The failure is loud, at least.
- **A superuser credential is now part of a deploy procedure.** ADR-0010 kept superuser out of
  application code and out of migrations, and both still hold. But installing the extension,
  and every future pgvector version upgrade (`ALTER EXTENSION vector UPDATE`), needs
  `POSTGRES_SUPER_PASSWORD` in an operator's hands, not just at instance creation.
- **The image tag tracks pgvector releases.** `pgvector/pgvector:pg17` moves with each pgvector
  release. Shared libraries and installed extension versions can drift apart across a pull.
  Pinning the pgvector version in the tag (`0.8.7-pg17`) is the obvious mechanism tightening.
  The backup image stays `postgres:17-alpine`. pg_dump doesn't need the extension's library,
  and the same-major rule in `deploy/backup/Dockerfile` still holds.
- **Exact scan is linear.** Each query costs one distance computation per candidate row. That
  is cheap while a user's rows plus the shared corpus number in the thousands, and it gets
  worse the more the shared corpus grows, because every user scans it.
- **Re-embedding at a different dimension is still a schema change.** The column is typed
  `vector(512)`. Per-row metadata covers a model change at the same dimension. A new dimension
  needs a new column or an untyped `vector` column, i.e. a real migration.
- **The `vector` type makes `fireplace_insights` the one database that can't be restored into a
  plain `postgres:17` instance.** ADR-0010 §6's extraction path still works, but the target must
  run the pgvector image and the bootstrap step.

**Revisit when:** a single user's filtered scan stops being cheap (latency or CPU measured, not
guessed). The next step is an HNSW index. Adding it must deal with the fewer-than-k effect:
pgvector 0.8's iterative index scans (`hnsw.iterative_scan`), a raised `hnsw.ef_search`, or a
partial index per corpus. That step can be an amendment. Moving vectors out of Postgres into a
dedicated store **supersedes** this ADR.
