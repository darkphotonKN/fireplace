---
id: I-KSJFR-6
status: in-progress
implements: FS-KSJFR
blocked_by: [I-KSJFR-1, I-KSJFR-2]
labels: [feature]
title: "FS-KSJFR slice 6: the day gate"
---
Implements FS-KSJFR §Requirements (R1–R8, R10, R15, R47–R51), §Edge States

## What to Build

The focus prompt stops being unconditional. This is the slice the whole feature was scoped
around: the prompt is kept and liked, but it asks once a day, and only when asking makes sense.

**The rule (R3).** The gate fires when **both** hold: no gate has fired yet during the current
browser-local calendar day, **and** nothing was *touched today* (I-KSJFR-1's stamp). The idle
condition is not decoration — the prompt *creates a plan*, so a plain daily cadence would ask a
user three weeks into a project to start something new every morning (D2).

**Decided before paint (R4).** From `localStorage` only, synchronously. The dashboard's data
fetch must not be able to change which state is shown; home must not open on a spinner while
the gate makes up its mind.

**One route, two states (R2).** `/` renders either the prompt or the dashboard. No redirect, no
second URL, no back-button trap.

**Firing is recorded (R5).** A second visit the same day goes straight to the dashboard whether
or not the user acted on the prompt. Skipping records that the gate fired but does **not**
record a touch — skipping is not activity (R10).

**Mount only (R6).** A tab left open across midnight keeps the state it has. No re-gate on focus
or visibility change. This is an accepted limit, not an oversight.

**Unchanged (R1, R7, R8).** Logged out, `/` is the product tour exactly as FS-0003 leaves it,
including the deliberate no-wait-on-`isLoading` behavior. The prompt's funnel is untouched:
plan type, focus line, start action to `/create-plan` with `name`, `focus`, `planType`. The
prompt keeps the greeting ("Welcome back, {name}", falling back to "there") and is the only
surface in this feature that greets by name (D8).

**Storage hostility (R15).** Absent, malformed or throwing storage reads as "not fired" and
"not touched": the gate errs toward showing the prompt, never toward an error.

## Acceptance Criteria

- [ ] Logged out, `/` is the product tour, unchanged.
- [ ] Nothing touched and no gate today: `/` shows the prompt with the name greeting.
- [ ] Gate already fired today: `/` shows the dashboard with no flash of the prompt.
- [ ] An item ticked earlier today anywhere in the app: `/` shows the dashboard.
- [ ] Only having viewed plans today does not suppress the prompt.
- [ ] A second visit the same day goes straight to the dashboard.
- [ ] Skipping records the gate as fired but does not record a touch.
- [ ] The decision is made without waiting on any network request.
- [ ] The prompt's start action still reaches `/create-plan` with name, focus and planType.
- [ ] Cleared or throwing `localStorage` shows the prompt rather than erroring.
- [ ] A tab open across midnight keeps its state until reload.
- [ ] Tests pass.

## Blocked By

I-KSJFR-1 — the stamp is the signal this reads.
I-KSJFR-2 — there must be a dashboard to show when the gate does not fire.

## Spec Reference

FS-KSJFR §Requirements (R1–R8, R10, R15), §Edge States (midnight, second device, cleared
storage), §Decisions and why (D1, D2, D3, D8, D12).

## TDD Approach

- RED: a table of gate cases — {gate fired today?} × {touched today?} — asserting prompt vs
  dashboard for each of the four combinations.
- GREEN: the gate evaluation on mount.
- Then: skip records fired-but-not-touched; storage-hostile reads fall to showing the prompt;
  the logged-out path still renders the tour.
