---
id: I-0037
status: done
implements: ADR-0009
blocked_by: []
labels: [enhancement]
title: "ADR-0009: remove dead services and unused infrastructure (example, orchestrator, Temporal, Makefile gap)"
---
Implements ADR-0009 §2, §6 and ADR-0011 §1

> **Anchor note.** This task spans two ADRs — ADR-0009 §2 (delete the scaffolds) and
> ADR-0011 §1 (delete Temporal). `implements:` carries the primary; both are named on
> this anchor line, which is what `code-review`'s spec axis reads. Grouped because every
> item is pure deletion with no behavior to preserve, and splitting them would produce
> three issues nobody can fail independently.

## What to Build

Pure removal. Nothing here changes behavior, because none of it has any.

**1. Fix the `Makefile` build audit first.** `SERVICES` currently reads
`api-gateway auth-service calendar-service example-service plan-service` — insights-service
has never been in it, so the platform's other event-flow participant is outside `build-all`,
`clean-builds` and `check-builds`. Set it to the post-consolidation list now, so the audit
covers the right things during the rest of the migration:

    SERVICES = api-gateway auth-service calendar-service insights-service plan-service

(auth and calendar drop out in I-0038, not here.)

**2. Delete `services/example-service` and `services/orchestrator-service`.** Both are verified
scaffolds: a single `Ping` RPC returning `"pong: <msg>"`, an AMQP topology with no bindings
behind a no-op consumer, and — for orchestrator — no database by design. There is no domain to
fold. Remove:

- the two service directories
- their `go.work` entries
- their compose services and databases (`example-service-db`, `example-service-test-db`) and
  the `fireplace_example_service_pgdata` / `fireplace_example_service_test_pgdata` volumes
- their rows in the root `SPECIFICATION.md` service map

The scaffold pattern they demonstrated lives in `GO_MCP_PROJECT_TEMPLATE.md` and
`setup-go-mcp-project.sh`; that is where a new service gets copied from. Nothing is lost.

**3. Delete Temporal from `docker-compose.yml`:** the `temporal`, `temporal-ui` and
`temporal-postgres` services plus the `temporal-pgdata` volume. No Go code references Temporal —
verified across `.go`, `.mod`, `.yml`, `.yaml`, `.toml`, `Makefile` and `.md`, where the only
hits are the compose definitions themselves and ADR-0008 §6, which ADR-0011 has already amended.
If a reference does turn up during the work, **stop and flag it** rather than deleting around it.

## Acceptance Criteria

- [ ] `make build-all` passes with insights-service included
- [ ] `make check-builds` passes
- [ ] `services/example-service` and `services/orchestrator-service` are gone, along with their
      `go.work` entries, compose services, databases and volumes
- [ ] `go work sync` is clean and the workspace builds
- [ ] `temporal`, `temporal-ui`, `temporal-postgres` and `temporal-pgdata` are gone from compose
- [ ] `docker compose up` brings the remaining stack up healthy
- [ ] Root `SPECIFICATION.md` no longer lists example-service or orchestrator-service
- [ ] `grep -ri temporal` over the repo returns only ADR-0008 and ADR-0011

## Blocked By

None. This is the first slice — it shrinks the surface everything else operates on.

## Spec Reference

ADR-0009 §2 (scaffolds deleted rather than folded), §6 (the `SERVICES` gap and why it is fixed
first); ADR-0011 §1 (Temporal removed from compose).
