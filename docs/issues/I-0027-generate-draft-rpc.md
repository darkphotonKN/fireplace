---
id: I-0027
status: open
implements: FS-0006
blocked_by: []
labels: [feature]
title: [HUMAN] FS-0006: planless GenerateDraft RPC and plan-type prompts
---

> **HUMAN-OWNED — do not run `/develop` on this issue.**
> Flagged in FS-0006 §Ownership split as the owner's lane. An agent must not
> implement it; pick it up only if the owner hands it over explicitly.

Implements FS-0006 §Requirements, §API surface

## What to Build

The server half of guided creation. The client half and the proto method land in I-0028; this is
the implementation behind them.

- **`GenerateDraft(seed, plan_type) → draft`** — **planless**. Every other method on this service
  begins by fetching Plan Context; this one cannot, because no plan exists yet. That is what makes
  it a distinct shape rather than another parameterization.
- **Plan-type prompts, one code path.** Learning and project plans **swap prompts**, they do not
  branch structurally. The data model is identical, and prerequisites-versus-dependencies is a
  content difference the LLM expresses through nesting and sequence — both of which two-tier
  nesting already carries.
- **Compact derivation vs full draft.** A compact derivation of the draft feeds high-frequency
  calls; the full draft is reserved for low-frequency deep calls. The draft is unbounded, so
  putting it in every prompt is both expensive and context-diluting.

## Acceptance Criteria

- [ ] The RPC takes no `plan_id` and calls no plan-service method.
- [ ] Learning and project seeds produce structurally different drafts from one code path.
- [ ] Generation failure surfaces as an error the caller can distinguish from unreachability.
- [ ] Once this ships, `POST /api/plans` with `mode=guided` stops returning 501.
- [ ] Tests pass.

## Blocked By

None. **Note:** I-0028 ships the guided path returning 501 `NOT_IMPLEMENTED` until this lands —
a declared delivery state, not a failure.

## Spec Reference

FS-0006 §Requirements R2, R12, R48 · §API surface (plane 2)
