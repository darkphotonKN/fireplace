---
id: I-02JBZ-16
status: done
implements: FS-02JBZ
blocked_by: [I-02JBZ-13]
labels: [feature]
title: "FS-02JBZ slice 16: the Day plan tooltip names the hours, and a cog changes them"
---
Implements FS-02JBZ §Change: Hours a day (§Requirements R114–R118)

## What to Build

Client only, in `client/src/components/DayPlanSection.tsx`.

- **Hours in words** (R115): a small formatter, 30 → "30 minutes", 60 → "1 hour", 90 → "1½ hours",
  … 720 → "12 hours".
- **Tooltip** (`DayPlanInfo`, R114): one line after the existing text, "Sized for {hours} a day.
  Change it with the cog." when set, "Sized for a typical day. Set your hours with the cog." when
  unset, from the current `settings` in state (updates at once after a save).
- **Cog** (R116): in the header beside Hide/Show, same glyph and button pattern as the Items
  card's settings button in `client/src/components/Todo.tsx`; shown only while the section is
  shown and the day plan has loaded. It swaps the section body for a settings view: "Back to day
  plan" (same back-arrow pattern as Todo's), one row **Hours a day** with a −/+ stepper (30 minutes
  to 12 hours, 30-minute steps) and a readout, "Use the default" to clear, and the line "Applies
  the next time you plan your day."
- **Stepper** (R117): unset reads "Typical day" with no "Use the default"; the first press of
  either button sets 2 hours; − disabled at 30 minutes, + at 12 hours. Each change saves
  optimistically through `setDayPlanSettings`, one request after another, the last value pressed
  winning; a failure reverts to the last saved value with a quiet "Couldn't save. Try again."
  Switch on `code`, never `detail`.
- Saving marks asked and never calls `planMyDay` (R118).

## Acceptance Criteria

- [ ] The tooltip shows "Sized for 2½ hours a day. Change it with the cog." for 150, the unset line
      for none, and updates after a save without a reload.
- [ ] The cog appears beside Hide/Show only when shown and loaded; it opens the settings view;
      "Back to day plan" returns to the plan.
- [ ] From unset, + (or −) sets 2 hours and saves 120; − stops at 30 minutes, + at 12 hours.
- [ ] "Use the default" saves `null`; the readout returns to "Typical day".
- [ ] Three fast presses send three saves in order and the stored value is the last one.
- [ ] A failed save reverts the readout and shows the quiet error.
- [ ] Saving from the cog never calls `planMyDay`; a later not-planned card shows no ask.
- [ ] Tests pass (`npm test`, `npm run lint`, `npm run build` in client).

## Blocked By

I-02JBZ-13 (`settings` on DayPlan and `setDayPlanSettings`). Not blocked by I-02JBZ-15, but it
edits the same component; run them one after the other, not in parallel.

## Spec Reference

FS-02JBZ §Change: Hours a day: R114–R118; Edge States "Set via the cog before ever planning",
"Use the default", "Rapid stepper presses", "Stepper save fails", "Section hidden", "Day plan read
failed".

## TDD Approach

- RED: a formatter table test (30, 60, 90, 150, 720), then a `DayPlanSection.test.tsx` case that
  the tooltip reads the unset line and, after the cog's + press, the "2 hours" line, with the
  mocked `setDayPlanSettings` called with 120.
- GREEN: the formatter, the tooltip line, the cog view and the stepper.

## Files expected to touch

- `client/src/components/DayPlanSection.tsx`
- `client/src/components/DayPlanSection.test.tsx`
