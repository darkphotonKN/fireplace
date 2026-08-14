# ADR-0006 — The remaining gateway surface is retrofitted in one batch, and the swaggo document is retired

Status: accepted
Date: 2026-08-14
Scope: root — governs the api-gateway HTTP surface and the client's consumption of it
Realized by: FS-0004
Amends: ADR-0002 §8 (adoption policy) — the grandfather clause is discharged, not deleted

## Context

ADR-0002 adopted the contract layer with a **serialize-on-touch** policy: an endpoint gets a
typed handler and a generated schema when a feature touches it, never before. That was the
right call at the time. It avoids a big-bang migration, keeps each change reviewable, and lets
the toolchain be proven on something small before it is trusted with everything. FS-0002
executed it on `GET`/`PATCH /api/users/profile` and the toolchain came out validated.

The policy has now produced the failure it was not designed to prevent. Nothing has touched
the remaining endpoints, so **`openapi.yaml` describes 2 of the gateway's 34 operations.**
The document is correct and nearly useless: a consumer cannot learn the API from it, the
docs page shows two rows, and the generated client covers one resource.

Three consequences make this worse than "incomplete":

1. **A second, hand-written document is still load-bearing.** `swaggo` remains mounted at
   `/swagger`, generated from `//@` annotations, describing the legacy majority. That is
   precisely the second-derivation drift the contract layer exists to eliminate — two
   descriptions of one API, each editable independently, capable of agreeing with each other
   while both having drifted from what ships. The contract layer cannot deliver its central
   guarantee while it is the *minority* description of the surface.

2. **The gates are near-vacuous.** Regenerate-and-diff, Spectral, and the breaking-change
   ratchet all run against a document covering 6% of the surface. They pass, they are green,
   and they are watching almost nothing.

3. **Serialize-on-touch prices the retrofit into whoever is unlucky.** A feature touching one
   checklist endpoint inherits the cost of serializing it, transcribing its transport types,
   and cutting the frontend over — work that has nothing to do with that feature. The policy
   converts a one-time migration into a permanent tax that lands unpredictably.

This was observed directly: the docs page was opened, only profile appeared, and a protected
operation was read as unprotected. A separate bug contributed (no `securitySchemes` was
declared, so no operation rendered a padlock — fixed in `7ece3e9`), but the durable problem is
that the document is empty enough to be misleading about the shape of the API.

## Decision

**1. The remaining 32 operations are serialized in one campaign**, sliced by route group, under
FS-0004. Serialize-on-touch is not repealed as a *principle* — it remains correct for any
endpoint added after this campaign. It is **discharged** for the existing surface: there is no
longer anything left to touch.

**2. Every retrofitted operation is behavior-preserving at the transport layer.** Paths, status
codes, and response bytes are transcribed from what ships today. Transport mirrors are
mechanical copies of the current wire format, not improvements. Any shape a reviewer wants
changed becomes its own feature spec, after this one.

**3. The `{statusCode, message, result}` envelope is removed per group**, in the same slice as
that group's frontend cutover. ADR-0003 already decided bare resources; this fixes *when* the
break lands, so no consumer ever observes a half-migrated group.

**4. The breaking-change ratchet cannot see these breaks, so they are verified manually.**
Envelope removal is not a diff against the previous document — those operations were not *in*
the previous document. Each group's slice records a before/after probe against a running
gateway as its evidence. Trusting the ratchet here would be trusting a gate to catch something
structurally outside its input.

**5. `swaggo` is retired in the final slice** — the `/swagger` mount, the `//@` annotations,
`docs/docs.go`, `make gen`, and the `swaggo`/`gin-swagger` dependencies. Not before: until the
last group is serialized it is the only description of what remains. After it, the generated
OpenAPI 3.1 document is the single description of the HTTP surface, and there is no second
place to edit.

**6. The gate gaps found during scoping are closed as part of this campaign**, because a
retrofit guarded by unproven gates is a retrofit with no guarantee:
   - `--fail-on ERR` becomes `--fail-on WARN`. `ERR` alone does not catch a field rename —
     demonstrated in barrowspire, where a rename exited 0 with three warnings.
   - A `gates-selftest` target is added and wired into `make gates`. `contract-fixtures/`
     already exists in full and nothing runs it, so no fireplace gate has ever been observed
     rejecting anything.
   - A `check-contract-auth` gate is added, asserting *runs auth middleware ⇒ declares
     security*, read from **Go source**. It must not infer "needs auth" from a documented 401:
     `signin` answers 401 for bad credentials while requiring no token at all.

## Consequences

**Accepted costs.**

- One large campaign of low-intellectual-content, high-volume work. Mechanical transcription
  at this scale is exactly where care fails, which is why §2 is enforced by round-trip tests
  over populated fixtures rather than by review.
- Shapes get frozen that arguably should have been designed differently — `GET /api/users`
  publishes an unbounded list; `GET /analytics/user/:userId` publishes a type whose handler
  currently returns 501. Freezing them is deliberate: a published wrong shape is fixable
  through the normal breaking-change path, whereas an unpublished one is invisible.
- Several groups break their response shape at once. This is a pre-1.0 internal API with a
  single first-party consumer that deploys together with it; the same reasoning ADR-0005 used
  to choose `strict` request validation applies here.

**What is gained.**

- One description of the HTTP surface, derived from code, with no second place to edit.
- The gates start guarding the whole surface instead of 6% of it — and, for the first time in
  this repo, are proven to reject something.
- The permanent, unpredictable retrofit tax on unrelated features disappears.

**What this does not decide.**

- Plane 2 (gRPC, `buf lint` / `buf breaking`) stays unwired. Out of scope, still an
  acknowledged gap.
- No downstream service's API changes. This ADR governs the edge only.
- Nothing about *which* shapes are right. Every transcription is as-is by §2; redesign is a
  later feature's job.

## Alternatives considered

**Keep serialize-on-touch and wait.** Rejected. It has had a full development cycle to
converge and moved from 0/34 to 2/34. The rate is set by unrelated feature traffic, not by
intent, and there is no mechanism by which it finishes.

**Serialize everything but keep the envelope.** Rejected. It would publish a contract that
contradicts ADR-0003 on day one and require a second breaking pass over the same 32
operations later — the same cost, paid twice, with a wrong contract published in between.

**Remove swaggo first, to force the issue.** Rejected. It would delete the only description of
whatever is not yet serialized, trading an incomplete document for no document.

**Hand-write OpenAPI 3.1 for the legacy routes to fill the document faster.** Rejected
outright. This is the exact failure mode — a doc written alongside the code rather than derived
from it — that ADR-0002 exists to prevent. A fast, wrong contract is worse than a slow, real
one, because it is trusted.
