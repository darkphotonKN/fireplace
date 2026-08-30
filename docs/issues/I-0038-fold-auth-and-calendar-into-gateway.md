---
id: I-0038
status: open
implements: ADR-0009
blocked_by: [I-0037]
labels: [enhancement]
title: "ADR-0009: fold auth-service and calendar-service back into api-gateway"
---
Implements ADR-0009 §1, §5

## What to Build

Fold both services into api-gateway as packages. Do them **one at a time**, auth first, with the
stack verified between them — the two folds are the same operation twice, not one big move.

**Per service:**

1. Move the domain into the gateway as an internal package (`internal/auth` already exists for
   middleware and the identity bridge; the auth-service domain lands alongside it without
   colliding).
2. **Merge the migration lineage into the gateway's `migrations/` directory, renumbered onto the
   end of the gateway's sequence.** This is the delicate part and the reason this issue comes
   before the database consolidation: after folding, the gateway owns exactly one lineage for
   exactly one database. Do not carry a second `schema_migrations` into the gateway.
3. Replace the gateway's gRPC client calls to that service with direct in-process calls.
4. Delete the service directory, its `go.work` entry, its compose service and databases, its
   volumes, and its row in the root `SPECIFICATION.md` service map.
5. Drop it from `Makefile` `SERVICES`.
6. Fold its `SPECIFICATION.md` capability lines into `services/api-gateway/SPECIFICATION.md`
   under the appropriate bounded context. The capabilities still exist — only their address
   changes — so the lines move, and their `[x]` state moves with them.

**Constraint that is not negotiable:** JWT verification stays shared in `common/auth`. Auth
folding into the gateway does not make identity a gateway-private concern — plan-service and
insights-service still verify tokens themselves under ADR-0001. **Do not copy the verifier into
the gateway.** If something makes that seem necessary, stop and flag it.

gRPC ordinals `7101` (auth) and `7104` (calendar) are freed by this work.

## Acceptance Criteria

- [ ] Every endpoint auth-service served is served by the gateway, with unchanged behavior
- [ ] Every endpoint calendar-service served is served by the gateway, with unchanged behavior
- [ ] The gateway's `migrations/` holds one lineage covering gateway + auth + calendar tables,
      applying cleanly from empty
- [ ] `common/auth` is unchanged and still the only JWT verifier; plan-service and
      insights-service still verify independently
- [ ] **The plan ↔ insights event flow is verified working after the auth fold, and again after
      the calendar fold** — `plan.created` → insight generated → `insight.generated` →
      checklist items materialized
- [ ] Both service directories, `go.work` entries, compose services, databases and volumes gone
- [ ] `Makefile` `SERVICES` reads `api-gateway insights-service plan-service`
- [ ] Root `SPECIFICATION.md` service map lists api-gateway, plan-service, insights-service, client
- [ ] Auth and calendar capability lines live in `services/api-gateway/SPECIFICATION.md` with
      their checkbox state preserved
- [ ] `make build-all` and `make check-builds` pass

## Blocked By

I-0037 — deleting the scaffolds and fixing the build audit first keeps this fold operating on the
smallest possible surface.

## Spec Reference

ADR-0009 §1 (auth and calendar fold back into the gateway), §5 (JWT verification stays shared in
`common/auth`). ADR-0001 governs the identity model that must survive unchanged.

## TDD Approach

- RED: the gateway's existing auth middleware tests (`internal/auth/middleware_test.go`) must stay
  green throughout — they are the regression net for the identity seam.
- GREEN: fold, then re-run the full suite plus the two-hop event flow before starting calendar.
