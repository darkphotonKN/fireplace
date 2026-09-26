---
id: I-0060
status: in-progress
implements: FS-none
blocked_by: []
labels: [bug]
title: "The plan calendar can't parse the wire format either — same defect as I-0059, different file"
---
Implements FS-none — a defect in behavior that predates the spec system. (`develop`'s anchor
gate expects FS-NNNN or ADR-NNNN, so this may need picking up by hand.)

## What's wrong

`client/src/lib/calendar.ts:52`:

```
const t = parse(s, "yyyy-MM-dd", new Date());
return isValid(t) ? t : null;
```

A strict date-fns format against a value the API sends as RFC3339. `"2026-03-16T00:00:00Z"`
does not match `"yyyy-MM-dd"`, so `isValid` fails and `parseISODate` returns `null`.

**Correction, made while fixing this (the premise above was too strong).** The calendar route
does *not* send RFC3339 today: `services/api-gateway/internal/gateway/calendar/local.go:72`
formats with `dateLayout = "2006-01-02"`, and `CalendarItemResponse` publishes `startDate` /
`dueDate` as a bare `string` with no `format: date-time` (`openapi.yaml:58`). There is one
registered calendar route and one formatter, so the calendar was **not** broken in production.

The defect was **latent, not live**: a strict parser that fails silently to `null`, sitting
behind an API whose sibling endpoint publishes the same domain value as RFC3339
(`ChecklistResp`, `plan/model.go:222`), with nothing holding the two spellings of the rule in
sync. One change to either serializer would have turned it live, silently. That is still worth
fixing, and it is a smaller claim than this issue was opened on.

I-0059 fixed the identical defect in `client/src/lib/itemDates.ts` and settled the rule there:
**the date portion is authoritative and the time and zone are ignored**, because these are
date-only domain values that the server happens to store as midnight UTC — resolving the
instant to a local date would shift every date back a day west of Greenwich.

Found while fixing I-0059; scoped out of it deliberately so that fix stayed reviewable.

## What to Build

- Make `parseISODate` accept both shapes under I-0059's rule, reusing `itemDates.ts`'s parser
  rather than spelling the rule a third time. Two files disagreeing about what a date is, is
  how this defect got duplicated.
- Fixtures in the real wire format. The current calendar tests pass because every fixture is
  hand-written date-only — the same blindness I-0059 had.
- Check the rendered result end to end, not just the parser: a green `parseISODate` unit test
  was never the gap.

## Acceptance Criteria

- [ ] `parseISODate` returns the correct calendar date for an RFC3339 input.
- [ ] It still handles `YYYY-MM-DD`.
- [ ] An item with RFC3339 dates renders a bar/chip in the calendar.
- [ ] The date rule is shared with `itemDates.ts`, not restated.
- [ ] Tests carry RFC3339 fixtures and pass under a non-UTC `TZ`.

## Blocked By

None. Lands cleanly after I-0059, whose parser it should reuse.

## Spec Reference

FS-none. Same root cause as I-0059; unrelated to FS-KSJFR, which does not render the calendar.
