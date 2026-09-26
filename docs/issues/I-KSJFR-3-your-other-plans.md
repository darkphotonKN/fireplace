---
id: I-KSJFR-3
status: done
implements: FS-KSJFR
blocked_by: [I-KSJFR-2]
labels: [feature]
title: "FS-KSJFR slice 3: Your other plans"
---
Implements FS-KSJFR §Requirements (R39–R42, R47–R51), §Edge States

## What to Build

The rest of the plans, one quiet line each, beneath Continue.

- **Disjoint from Continue (R39).** The three plans shown as cards are excluded. No plan appears
  twice on the page — that was a deliberate call (D6), because on a four-plan account a repeated
  list reads as a rendering bug.
- **Capped at six rows with a link out to `/myplans` (R40).** Rows carry name and type only.
  Progress is deliberately absent: computing it would need an item fetch per plan and break
  ADR-0013's cap.
- **Absent at three plans or fewer (R41).** The block does not render with an empty body and
  does not render a heading over nothing.
- **Read-only (R42).**

Visual conformance per R47–R51: tokens only, neutral — the coral moment is not here.

## Acceptance Criteria

- [ ] With 10 plans, Continue shows 3 and this block shows 6 with a link to `/myplans`.
- [ ] No plan appears in both blocks.
- [ ] With exactly 4 plans, this block shows 1 row and no "all plans" link is needed.
- [ ] With 3 plans, the block is absent — no heading, no empty state.
- [ ] With 0 plans, the block is absent (slice 7 owns what shows instead).
- [ ] Rows show name and type; no per-plan item request is issued.
- [ ] Request count is unchanged — still `1 + min(planCount, 3)`, four at 10 plans.
- [ ] Tokens only; both themes correct; no coral in this block.
- [ ] Tests pass.

## Blocked By

I-KSJFR-2 — this renders inside the shell and consumes the same fetched plan list.

## Spec Reference

FS-KSJFR §Requirements (R39–R42), §Edge States (one plan, zero plans), §Decisions and why (D6).

## TDD Approach

- RED: a test at 10 plans asserting the six rows are the 4th–9th by `updatedAt` and that none of
  them is a Continue card.
- GREEN: the derivation and the list.
- Then: the 3-plan absence case and the request-count assertion.
