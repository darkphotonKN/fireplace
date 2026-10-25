---
id: I-02JBZ-9
status: done
implements: FS-02JBZ
blocked_by: [I-02JBZ-6, I-02JBZ-8]
labels: [feature]
title: "FS-02JBZ slice 9: day ticks in the Day plan section, with the inline \"Also mark done?\""
---
Implements FS-02JBZ §Requirements (R75–R79, R82, R83)

## What to Build

The day tick as the section's primary checkbox (D4).

- **Tick / untick (R75).** A tickable entry has a checkbox calling `tickDayPlanEntry` /
  `untickDayPlanEntry`, optimistic, reverted on failure.
- **Added steps (R76).** Right after Add, tick with the created item's id. After a reload, an
  added `new_step` entry is recognised by a child of its parent whose description matches the
  entry text (case/whitespace-insensitive) and ticked with that child's id. Put the matcher in
  `lib/dayPlan.ts`.
- **"Also mark '<item>' done?" (R77, S12).** After a tick, when the item is not already done,
  an inline link beside the entry; dismisses after 8 seconds or on any other action; **Yes**
  sets `done` through the existing item update (optimistic, reverted on failure). No modal.
  Unticking removes it.
- **Independent (R78).** The plan view's own tick keeps working; a day tick never changes `done`
  and `done` never changes the day tick.
- **Not tickable (R79).** `400 · DAY_PLAN_ENTRY_NOT_TICKABLE` reverts quietly and refetches the
  day plan.
- **Survives redraws (R82).** Ticked entries stay, shown ticked, for the day and through Plan
  again. **Rules plans (R83)** tick the same way.

## Acceptance Criteria

- [ ] Ticking and unticking are optimistic and revert on failure; `done` is untouched by either.
- [ ] A just-added step is tickable at once; after a reload it is recognised and tickable.
- [ ] After a tick on an item not yet done, "Also mark '<item>' done?" appears inline, disappears
      after 8 seconds or on another action, and Yes sets `done` (reverting on failure); no link
      when the item is already done.
- [ ] `DAY_PLAN_ENTRY_NOT_TICKABLE` reverts and refetches, switching on `code`.
- [ ] Ticked entries survive Plan again; basic-plan entries tick.
- [ ] Tests pass.

## Blocked By

- I-02JBZ-6 — tick/untick operations and `ticked`.
- I-02JBZ-8 — the section.

## Expected write set

- `client/src/components/DayPlanSection.tsx` (+ test) **(hotspot: shared with I-02JBZ-10)**
- `client/src/lib/dayPlan.ts` (+ test)
- `client/src/api/insights.ts` **(hotspot)**: `tickDayPlanEntry`, `untickDayPlanEntry`
- `client/src/api/checklists.ts` (only if the existing update helper needs a `done`-only call)

## Spec Reference

FS-02JBZ §Decisions D3, D4; §Requirements R75–R79, R82, R83; §Spec defaults S5, S12; §User Stories
12–18; §Edge States ("Entry's item ticked done in the plan view", "New step added, then page
reloaded", "New step added, text edited before ticking", "Shared plan").

## TDD Approach

- RED: component test: tick → optimistic checked, `tickDayPlanEntry` called, item update not
  called; failure → reverted.
- GREEN: the checkbox.
- Then: the inline ask (fake timers for 8 s; Yes → update with `done: true`; already-done → no
  link); added-step id on tick and the reload matcher; not-tickable revert and refetch.
