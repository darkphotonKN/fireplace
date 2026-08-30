---
id: I-0039
status: open
implements: ADR-0010
blocked_by: [I-0038]
labels: [enhancement]
title: "ADR-0010: three databases on one Postgres instance, with DDL split out of the runtime role"
---
Implements ADR-0010 §1, §2, §4, §5, §6

## What to Build

Collapse the remaining Postgres containers onto one instance and take DDL away from the runtime
roles. Do it locally first, in this order, verifying between the two halves.

**Half one — topology.** One production Postgres container holding three databases:

| Database | Owner service |
|---|---|
| `fireplace_gateway` | api-gateway (incl. the folded auth and calendar tables) |
| `fireplace_plans` | plan-service — plans, checklist items, outbox |
| `fireplace_insights` | insights-service — insights, `processed_events`, outbox when it lands |

**Three databases, not three schemas.** PostgreSQL has no cross-database query, so isolation is
enforced by the planner rather than by grants and `search_path` discipline. Each database keeps
its own `public.schema_migrations`, so the three lineages stay independent with no version-table
configuration. `config/db.go` does not change for this half — only `DB_NAME`, `DB_HOST` and
`DB_PORT` in each service's `.env`.

Mirror the same shape for tests: one test instance, three test databases. Tests then exercise
the real grant model instead of routing around it.

**Half two — roles and migration execution.** Two roles per database:

- `<db>_owner` — owns the database, holds DDL, used **only** by the migration step
- `<db>_app` — `USAGE` plus `SELECT`/`INSERT`/`UPDATE`/`DELETE`, **no `CREATE`**, the only role a
  running service connects as

This requires lifting `runMigrations` out of `InitDB` (`services/plan-service/config/db.go:46`
and the identical shape in the other services) into a discrete step that runs to completion and
exits before the application starts. The mechanism — init container, `make migrate`, or a
pipeline stage — is an implementation choice; **whatever is chosen must have a defined outcome
when a migration fails partway**, and that outcome belongs in the PR description.

While in here: `file://migrations` is a relative path resolved against the working directory.
Moving migrations out of the app process changes that working directory and it will need fixing
at the same time.

**Do not touch** the outbox, relay or inbox implementations. insights-service's outbox does not
exist yet and this work must not constrain its design.

## Acceptance Criteria

- [ ] One production Postgres container serving `fireplace_gateway`, `fireplace_plans`,
      `fireplace_insights`; one test container mirroring it
- [ ] All previous per-service Postgres containers and their volumes are gone from compose
- [ ] Each service connects as `<db>_app`, and `CREATE TABLE` from that role **fails**
- [ ] Migrations run as `<db>_owner` in a step that completes and exits before the app starts
- [ ] Each database has its own `public.schema_migrations` with its own version sequence
- [ ] plan-service and insights-service are in **different databases** — verify a cross-database
      query is impossible, not merely disallowed
- [ ] The plan ↔ insights event flow works end to end after the move
- [ ] The full test suite passes against the mirrored test tier
- [ ] `file://migrations` resolves correctly from the new migration step's working directory
- [ ] No outbox, relay or inbox source file is modified by this issue

## Blocked By

I-0038 — auth and calendar must already be folded, or `fireplace_gateway` would receive three
independent migration lineages from three separate processes, which is precisely the silent
version-skip failure ADR-0010 exists to prevent.

## Spec Reference

ADR-0010 §1 (three databases, one instance), §2 (owner/app roles), §4 (migrations out of
`InitDB`), §5 (test tier mirrors production), §6 (extraction stays mechanical). §3 — plan and
insights never share a database — is the hard constraint this must not violate.
