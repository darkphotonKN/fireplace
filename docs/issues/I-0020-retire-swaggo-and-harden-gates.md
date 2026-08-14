---
id: I-0020
status: open
implements: FS-0004
blocked_by: [I-0016, I-0017, I-0018, I-0019]
labels: [ready-for-agent]
---
Implements FS-0004 §Requirements, §Acceptance Criteria

## What to Build

The closing slice. Once every group is serialized, remove the second description of the API
and make the gates actually guard it.

**Retirement (R20)** — only after the last group lands, because until then swaggo is the only
description of whatever remains:

1. Remove the `/swagger/*any` mount from `config/routes.go`.
2. Delete `services/api-gateway/docs/` (`docs.go` and the generated 2.0 document) and the
   blank import that registers it.
3. Strip the `//@` annotation blocks from the handlers.
4. Remove the `make gen` target and the `swaggo`/`gin-swagger` dependencies from `go.mod`.

**Gate hardening (R21–R23)** — each of these closes a gap where a gate currently passes
without doing anything:

5. `--fail-on ERR` → `--fail-on WARN` in the ratchet target
   (`services/api-gateway/Makefile:233`). `ERR` alone does not catch a field rename —
   demonstrated in barrowspire, where a rename exited 0 with three warnings. After changing
   it, confirm no false positives against an unchanged document.
6. Add a `gates-selftest` target that runs the existing `contract-fixtures/` and wire it into
   `make gates`. Two traps, both of which produced a vacuous selftest in barrowspire:
   - **Verify the file globs match real filenames.** The fixtures here are
     `spectral-bad.yaml`, `oasdiff-base.yaml`, `oasdiff-rev.yaml`,
     `oasdiff-allowlist.txt` — a glob like `*.bad.yaml` matches nothing and the selftest
     passes vacuously.
   - **Assert on exit status, not on grepping output for "error".**
7. Add a `check-contract-auth` gate: every operation that runs `auth.AuthMiddleware` must
   declare `Security`. Read this from **Go source**, not from the generated document.
   **Do not infer "needs auth" from a documented 401** — `signin` answers 401 for bad
   credentials while requiring no token at all. That inference is what broke the first version
   of this gate in barrowspire.
8. Add the frontend lint fence: ban bare `fetch` outside the single client module, with an
   override for that module. Land it here rather than earlier because the tree only becomes
   clean once every group has been cut over.

**Proving the gates.** Each gate above must be **observed rejecting something**, and the run
recorded in the close-out. A gate that cannot distinguish *passed* from *did not run* emits a
green check either way, which is worse than no gate because it is trusted. Specifically:

- remove one `Security` line → `check-contract-auth` names the operation and exits non-zero;
  restore it → clean
- rename a field → the ratchet fails under `WARN` (and would have passed under `ERR`)
- the lint fence rejects a fixture file and accepts the tree

## Acceptance Criteria

- [ ] `/swagger` returns 404; `docs/docs.go` is deleted; `make gen` is gone
- [ ] `swaggo` and `gin-swagger` are absent from `go.mod` and `go.sum`
- [ ] No `//@` annotation blocks remain in the gateway
- [ ] The ratchet runs with `--fail-on WARN` and produces no false positives on an unchanged
      document
- [ ] A field rename is **observed failing** the ratchet, with the recorded run in the
      close-out
- [ ] `gates-selftest` is wired into `make gates`, its globs match real filenames, and it
      asserts on exit status
- [ ] Each fixture is **observed being rejected** by the gate it targets
- [ ] `check-contract-auth` passes on the tree, and is **observed failing** when one `Security`
      declaration is removed
- [ ] The lint fence rejects a fixture and accepts the tree, with the recorded run
- [ ] No raw `fetch`/`axios` call to a gateway path remains outside the client module
- [ ] `/api/docs` renders all 34 operations with padlocks on the protected ones
- [ ] `make gates` green end to end; full Go test suite and client typecheck green

## Blocked By

I-0016, I-0017, I-0018, I-0019 — swaggo cannot be removed while it is still the only
description of an unserialized group.

## Spec Reference

FS-0004 §Requirements R18, R20–R23 · §Acceptance Criteria

## TDD Approach

Gate-shaped work, so the loop is inverted: the fixture is the test.

- RED: point each gate at its known-bad fixture, confirm it **fails**
- GREEN: point it at the real tree, confirm it passes
- A gate that passes both is not wired up — diagnose it rather than accepting the green
