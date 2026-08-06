---
id: I-0001
status: open
implements: ADR-0002
blocked_by: []
labels: [enhancement]
title: Implements ADR-0002: contract gate tooling (oasdiff + allowlist, Spectral ruleset, regenerate-and-diff CI, Makefile openapi/client targets)
migrated_from: github#46
---
Implements ADR-0002 — contract layer: two planes, design-first process with code-first generation

> **Anchor note.** This task is anchored to an **ADR**, not a feature spec. It is
> ADR-mandated *infrastructure* — there is no user-visible capability to specify, so it
> correctly has no FS. `Implements ADR-NNNN` is a legitimate second anchor form alongside
> `Implements FS-NNNN §<section>`. If `develop`'s entry gate or `code-review`'s spec axis
> rejects a non-FS anchor, that is a **skill patch** (accept ADR-implementation as an anchor
> type for infrastructure tasks), not a reason to invent a feature spec for this work.

## What to Build

The contract gates ADR-0002 mandates but does not itself stand up. FS-0002 *consumes* these
gates and explicitly excludes building them (see FS-0002 § Out of Scope).

- **`oasdiff`** — breaking-change detection against the committed `openapi.yaml`. `ERR`
  blocks merge.
- **`.oasdiff.yaml` allowlist** — deliberate breaks pass only via an explicit, reviewed
  entry with a stated reason.
- **`.spectral.yaml` ruleset** — contract depth: descriptions, examples, and error schemas
  required on every operation.
- **CI regenerate-and-diff** — regenerate the spec and fail if the committed artifact
  differs from the derived one.
- **Makefile targets** — `openapi` (regenerate the spec) and `client` (regenerate the TS
  client). These are the commands `develop`'s pre-flight contract check invokes; today that
  check assumes a regeneration command exists without naming one.

## Acceptance Criteria

- [ ] `oasdiff` runs in CI against the committed `openapi.yaml`; a breaking change fails the
      build.
- [ ] `.oasdiff.yaml` exists; an allowlisted break passes with its reason recorded.
- [ ] `.spectral.yaml` exists and fails an operation missing a description, example, or
      error schema.
- [ ] CI regenerates the spec and fails when committed != regenerated.
- [ ] `make openapi` and `make client` exist and are idempotent.
- [ ] Running the full gate set against the current (pre-serialization) tree passes or fails
      for stated, understood reasons — no silent skips.

## Blocked By

None. Sequence **parallel to FS-0002 slices 1–2**.

## Spec Reference

ADR-0002 (as amended by ADR-0004) — plane 1 gates. FS-0002 § Out of Scope names this work as
deliberately excluded from that feature.

## Sequencing note

**Must land before FS-0002 slices 3–5 can close their gate-dependent acceptance criteria** —
specifically: "oasdiff reports the envelope removal and the 401 shape change as the only
breaking changes, and both are present in the allowlist", "regenerating the spec produces no
diff against the committed openapi.yaml", and "Spectral passes".

## Known blocker to resolve here

`services/api-gateway/go.sum` is currently **gitignored**, which undercuts the
regenerate-and-diff gate's assumption of a pinned, verifiable module graph. Fix as part of
this task.

## Extraction note (SSOT)

Every artifact this task produces — `contract.yml`, `.oasdiff.yaml`, `.spectral.yaml`, the
Makefile targets — is **extraction-bound for `setup/templates`**. In template-land this
entire task collapses into "setup copies these files," which is precisely why it never
deserved a feature spec. See `docs/notes/contract-pioneer-log.md`.
