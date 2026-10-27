---
id: I-DF4ZB-1
status: in-progress
implements: FS-DF4ZB
blocked_by: []
labels: [feature]
title: "FS-DF4ZB slice 1: pgvector in fireplace_insights, rag_chunks, the embedder and its repository"
---
Implements FS-DF4ZB §Requirements (R31–R34, R31a–R31d), §Edge States

## What to Build

The storage floor every retrieval slice stands on, per ADR-0018. Nothing reads it yet: the
retriever is H2 (I-DF4ZB-14) and ingest is H3 (I-DF4ZB-18) and the playbook command
(I-DF4ZB-12). This slice makes the table exist, the extension available without any app role
gaining superuser, the backups survive it, and gives the rest of the feature one embedder and one
repository to call.

**Infrastructure (R31a–R31d).**

- **Image (R31a).** The shared Postgres service in `docker-compose.yml` and `postgres-test` in
  `docker-compose.override.yml` both move to `pgvector/pgvector:0.8.7-pg17`, an exact tag, in the
  same change. The backup image stays `postgres:17-alpine`.
- **Bootstrap (R31b).** `scripts/db/init-databases.sh` gains a superuser step **outside** the
  `SET ROLE <owner>` block that runs `CREATE EXTENSION IF NOT EXISTS vector` in
  `fireplace_insights` only. It creates the extension and nothing else. `fireplace_plans` and
  `fireplace_gateway` never get it. Runtime role gains no `CREATE`.
- **Existing volumes (R31c).** A documented one-off, beside the backup and restore-drill runbook
  in `deploy/README.md`: the exact `CREATE EXTENSION vector;` command per environment (prod, local
  dev, test), run as the instance superuser, before the `rag_chunks` migration reaches that
  volume. Run it on local dev and test as part of this slice. **The prod run is the owner's to
  execute** (FS §Ownership split, agent lane note); the slice ships the documented command, not
  the prod run.
- **Backup and restore (R31d).** `deploy/backup/backup.sh` dumps `fireplace_insights` with
  `pg_dump --exclude-extension=vector`. `deploy/backup/restore.sh` states and checks its
  precondition (the target `fireplace_insights` already has `vector`) and refuses with a clear
  message otherwise. Perform one restore of a `fireplace_insights` dump that contains
  `rag_chunks` rows into a target that has the extension and show the rows are present and
  queryable with `<=>`. The existing restore drill stays green.

**The table (R31–R33).** A new numbered migration in `services/insights-service/migrations/`
creating `rag_chunks` with the columns of R31, `(source, source_id, chunk_no)` unique, no ANN
index. The migration assumes the extension exists and never creates it.

**The embedder (R34).** One component calling `text-embedding-3-small` with `dimensions=512`,
stamping `embedding_model` and `embedding_dims` on every row it produces. Ingest and query both
go through it. Keep its surface narrow (text in, vector plus model/dims out); H2 and H3 build on
it.

**The repository (R32).** A plain upsert keyed by `(source, source_id, chunk_no)`: it stores
`content_hash` with the row but does **not** compare it and never skips. A second upsert with the
same key replaces the row. The compare-and-skip on re-ingest (an unchanged chunk is neither
re-embedded nor rewritten) and the removed-document sweep for playbook rows belong to H4
(I-DF4ZB-12, HUMAN), not this slice. Plus the delete-by-source and "does this user have any
personal rows" reads that later slices call (I-DF4ZB-5's gate, H4's sweep, H3's
un-completion). **No similarity query here**: the query
shape is H2's (R26–R27).

## Acceptance Criteria

- [ ] The shared instance and `postgres-test` both run `pgvector/pgvector:0.8.7-pg17`; the backup
      image is still `postgres:17-alpine`.
- [ ] On a fresh volume, `init-databases.sh` creates `vector` in `fireplace_insights` as superuser,
      outside the `SET ROLE` block; `fireplace_plans` and `fireplace_gateway` have no `vector`.
- [ ] No migration contains `CREATE EXTENSION vector`; app and migration roles connect as their
      own non-superuser roles; the runtime role has no `CREATE`.
- [ ] The one-off procedure for existing volumes is documented per environment (prod, local dev,
      test) and has been run on local dev and test before the migration lands there.
- [ ] `rag_chunks` exists with the R31 columns, the R32 unique key, and no ANN index.
- [ ] `backup.sh` dumps `fireplace_insights` with `--exclude-extension=vector`; `restore.sh`
      refuses a target without the extension.
- [ ] A restore of `fireplace_insights` containing `rag_chunks` rows has been performed into a
      target with the extension; rows are present and queryable with `<=>`; the drill is green.
- [ ] Every row the embedder produces records `text-embedding-3-small` and 512.
- [ ] A second upsert with the same `(source, source_id, chunk_no)` replaces the row: one row,
      not two.
- [ ] Tests pass.

## Blocked By

None.

## Expected write set

- `docker-compose.yml`, `docker-compose.override.yml` (image pins)
- `scripts/db/init-databases.sh`
- `deploy/backup/backup.sh`, `deploy/backup/restore.sh`, `deploy/README.md` (R31c runbook)
- `services/insights-service/migrations/00000N_create_rag_chunks_table.{up,down}.sql` (next free
  number at develop time; see watch note below)
- `services/insights-service/internal/rag/` (new: embedder, repository, model, tests)
- `services/insights-service/.env.example` (embedding model/dims config, if configurable)
- `services/insights-service/config/services.go` **(hotspot)**, only if the embedder is
  constructed there in this slice

Watch: `services/insights-service/migrations/` is shared with I-DF4ZB-4, -5, -6 and -10, which
can run in parallel with this slice. Take the next free number at develop time and rebase on a
collision.

## Spec Reference

FS-DF4ZB §Requirements R31–R34 and R31a–R31d; §Decisions L8; §Edge States ("Existing volume never
given the extension", "Restore into a target without the extension"); ADR-0018, ADR-0010 §2 and
§5, ADR-0012 §6.

## TDD Approach

- RED: a repository integration test (against `postgres-test` on the new image) that upserts
  twice with the same `(source, source_id, chunk_no)` and different content, and asserts one row
  holding the second write.
- GREEN: migration plus upsert.
- Then: the embedder stamps model/dims; `restore.sh` refusal on a target without `vector`.
