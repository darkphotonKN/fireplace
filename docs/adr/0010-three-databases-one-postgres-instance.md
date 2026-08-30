# ADR-0010 — Three databases on one Postgres instance, with DDL split out of the runtime role

Status: accepted
Date: 2026-08-30
Scope: root — governs every service's database connection, role grants and migration execution
Related: ADR-0009 (the three services these databases belong to), ADR-0008 (the effectively-once
guarantee that depends on Decision 3), ADR-0012 (the hosting posture that forced the cost question)

## Context

Compose currently runs **ten Postgres containers** — one production and one test database per
service (`5301`-`5306`, `5556` for production; `6301`-`6306` for test). On a laptop that is
merely noisy. On the single VPS of ADR-0012, at 1-2GB of RAM, it is not affordable: each
instance carries its own `shared_buffers`, WAL writer, autovacuum launcher and background
worker set, and ten of them spend most of the box on overhead rather than data.

The requirement is to collapse them while keeping the isolation that makes ADR-0008's
choreography meaningful. Three layouts were on the table, and the wording that arrived with the
brief was ambiguous between them — "three databases on one box" and "three schemas on one
instance" describe materially different systems.

| | Isolation | Migration work | Overhead |
|---|---|---|---|
| Three **schemas**, one database | Soft — a qualified name crosses it | High | Lowest |
| Three **databases**, one instance | **Hard — the planner cannot cross it** | Near zero | Low |
| Three **instances**, one box | Hard | Zero | ~3× |

The schema layout was the original proposal and is the one that loses. It costs *more*
implementation work than separate databases and delivers *weaker* isolation — a strictly
dominated option, which is rare enough to be worth writing down.

**Why it costs more.** Three services migrating into one database all resolve `search_path` to
`public` by default, so `golang-migrate` places its version table — `schema_migrations` — in the
same place for all three. plan-service boots and stamps version 20; insights-service boots,
reads 20, concludes its own migrations 1 and 2 are already applied, and skips them. Silently.
No error, no log line. `processed_events` never gets created and the failure surfaces at the
first event delivery, in precisely the code path where a missing table means undetected
duplicate processing. Avoiding that needs a per-schema `x-migrations-table`, a per-role
`search_path` (because unqualified `CREATE TABLE plans` otherwise lands in `public` alongside
everyone else's tables), a grant matrix, and edits to `config/db.go` in all three services.

**Why it delivers less.** Even with all of that in place, the isolation is a `search_path`
setting and a grant. One qualified table name — `insights.generated_insights` — crosses it, and
the only thing standing in the way is review discipline at the moment someone wants a
convenient join.

Separate databases need none of that machinery and cannot be crossed at all. PostgreSQL has no
cross-database query: the planner has no mechanism for it. Reaching into another service's data
requires installing `dblink` or `postgres_fdw` first, which is a visible line in a migration
that nobody adds by accident.

Three separate instances would also be uncrossable, but pays roughly three times the memory
overhead for isolation that separate databases already provide in full.

Recorded without adversarial review — locked collaboratively during the consolidation scoping
session and not run through `challenge-me`.

## Decision

**One PostgreSQL instance, three databases, two roles each, and migrations that run outside the
application process.**

1. **Three databases on one instance:** `fireplace_gateway` (auth, calendar and whatever else
   folds in under ADR-0009), `fireplace_plans` (plans, checklist items, outbox),
   `fireplace_insights` (insights, `processed_events`, and the outbox when it lands). Not three
   schemas in one database.
2. **Two roles per database.** `<db>_owner` owns the database and holds DDL; it is used *only*
   by the migration step. `<db>_app` gets `USAGE` plus `SELECT`/`INSERT`/`UPDATE`/`DELETE` and
   **no `CREATE`**; it is the only role the running service ever connects as. No application
   code connects as a superuser.
3. **plan-service and insights-service must never share a database.** This is the one hard
   constraint. ADR-0008's effectively-once processing rests on the producer and the consumer
   being unable to enlist in the same transaction; putting them in one database makes that
   sharing possible, and the first person to notice will write the join that turns the event
   flow into decoration around a shared table. Same instance is fine. Same database is not.
4. **Migrations move out of `InitDB` into a discrete step** that runs to completion and exits
   before the application starts. Today every service calls `runMigrations` from inside
   `InitDB` on its own connection (`services/plan-service/config/db.go:46` and the same shape in
   each of the others), which requires the *runtime* role to hold `CREATE` permanently. That is
   incompatible with Decision 2, and it is the change Decision 2 actually costs.
5. **The test tier mirrors production exactly:** one test instance, three test databases, the
   same two-role split, the same migration step. Tests then exercise the grant model instead of
   routing around it, so a permission or role bug fails in CI rather than on first deploy.
6. **Extraction stays mechanical.** Moving a service to its own instance later is
   `pg_dump -d fireplace_plans`, restore into the new instance, change `DB_HOST`. No schema
   rename, no code change, provided the roles above were in place from the start.

## Consequences

**Good**

- Isolation is enforced by the query planner rather than by grant hygiene and code review. There
  is no qualified name that reaches across, because there is no such thing as a cross-database
  query.
- **`config/db.go` is untouched in every service.** Each service's `DB_NAME` changes and
  `DB_HOST`/`DB_PORT` converge on one instance — `.env` edits, not Go edits. The one Go change is
  lifting `runMigrations` out of `InitDB`, which is the same small edit three times.
- Each database keeps its own `public.schema_migrations`, so the three migration lineages stay
  independent by construction. No version-table configuration, and the silent-skip failure mode
  described above cannot occur.
- A compromised or injected application process can no longer `DROP TABLE`. The runtime role has
  no DDL. This is the point of Decision 2, and it is cheap now: the grant script is being written
  regardless, and a second role is three more lines.
- Boot no longer races on migrations. The current in-process approach survives one replica and
  relies on `golang-migrate`'s advisory lock beyond that; insights-service is precisely the
  service ADR-0012 expects to scale out first, which is the worst moment to discover this.

**Bad / accepted**

- **Shared failure domain.** One instance down is all three services down. Separate containers
  bought independent failure and this gives it up deliberately for cost. This is the single
  largest thing being traded away.
- **Shared tuning and shared resources.** One `shared_buffers`, one `work_mem`, one autovacuum
  budget serving three workloads with different shapes — insights-service's bursty LLM-driven
  writes against plan-service's steady transactional load. There is no way to tune for one
  without affecting the others.
- **Connection ceiling is now a real number.** Three services at `SetMaxOpenConns(25)` is 75
  connections against a default `max_connections` of 100, on one instance that also serves
  migrations and any psql session. It fits, with little room. Raising `max_connections` or
  introducing PgBouncer is a foreseeable follow-up rather than a surprise.
- **Migrations are now a deploy step that can be forgotten or can fail on its own.** The current
  boot-time approach has one moving part and no window in which code is deployed but migrations
  are not. Splitting them buys the security and replica properties above and costs exactly that
  simplicity. How the step is invoked — init container, `make migrate`, or a pipeline stage — is
  a mechanism detail left to implementation, but the failure semantics of a half-applied
  migration need an answer before the first production deploy, not after.
- **Two credential sets per database instead of one**, six in total, and they need somewhere to
  live that is not the repository (ADR-0012).
- **`file://migrations` is a relative path.** It resolves against the working directory, which is
  currently the service directory under `air`. Moving migrations to a separate step or into a
  container changes that working directory, and this will need fixing at the same time.
