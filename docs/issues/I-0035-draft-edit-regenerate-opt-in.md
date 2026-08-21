---
id: I-0035
status: open
implements: FS-0006
blocked_by: [I-0021, I-0032]
labels: [feature]
title: FS-0006 slice 11: draft-edit regenerate opt-in
---

Implements FS-0006 §Requirements

## What to Build

Editing `plan_draft` on a plan that already has items triggers **nothing automatically**. The draft
saves; the plan then offers *"draft changed — regenerate items?"* as an opt-in that routes through
the retry path.

Auto-regeneration would burn a paid LLM call on every save and delete work the user never asked to
lose. When the user does opt in, ADR-0007 applies as normal: new items are added, `generated` items
may be replaced, and `authored` / `touched` items are untouchable.

Editing the draft **while** items are generating is allowed — the in-flight generation used the
draft as it was at request time, and the offer appears once the plan leaves `generating`.

## HITL

Prompt copy and placement are not specified. Decide with the owner — in particular whether the
offer is persistent or dismissible, and whether it reappears on a later edit.

## Acceptance Criteria

- [ ] Saving a draft edit issues no generation and writes no outbox row.
- [ ] The offer appears only when the plan already has items.
- [ ] Accepting routes through the retry path, inheriting its cooldown and cap.
- [ ] Regeneration leaves `touched` and `authored` items byte-identical.
- [ ] Editing during `generating` is allowed; the offer appears afterwards.
- [ ] Tests pass.

## Blocked By

I-0021, I-0032.

## Spec Reference

FS-0006 §Requirements R56 · ADR-0007
