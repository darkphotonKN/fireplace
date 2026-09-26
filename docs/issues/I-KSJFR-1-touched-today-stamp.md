---
id: I-KSJFR-1
status: in-progress
implements: FS-KSJFR
blocked_by: []
labels: [feature]
title: "FS-KSJFR slice 1: record touched-today at the API layer"
---
Implements FS-KSJFR §Requirements (R11–R15)

## What to Build

The signal the day gate will read in slice 6. No UI in this slice — it is the foundation, and
it ships proved rather than assumed.

A small module (`client/src/lib/touchedToday.ts` or equivalent) owning the whole of this
concern:

- `markTouchedToday()` — writes today's browser-local date.
- `wasTouchedToday()` — true only when the stored date is today, local time.
- A matching pair for whether the gate has already fired today (slice 6 consumes it; storing
  both here keeps one module owning the storage keys).

**Where it is called from matters more than what it does.** Per R12 the stamp is written in
`client/src/api/*.ts` mutating functions — the single choke point every mutation already
passes through — not from components. Cover: plan create / update / delete /
toggle-daily-reset, and checklist item create / update / dates / reorder / archive / delete.

Dates are browser-local throughout. `client/src/lib/itemDates.ts` documents why
(`new Date("2026-03-09")` is midnight UTC, i.e. the previous day west of Greenwich) — follow
that file's local-time discipline rather than inventing a second convention.

Reads record nothing (R13): list, get, search, and the profile calls stay untouched.

A rejected request records nothing (R14) — the write happens only after the call resolves
successfully.

Storage is best-effort (R15): absent, malformed, or a throwing accessor (private mode, blocked
site data) all read as "not touched" and "gate has not fired". This module never throws into a
render path.

## Acceptance Criteria

- [ ] A successful plan mutation records touched-today.
- [ ] A successful checklist item mutation records touched-today, for each of create, update,
      dates, reorder, archive and delete. (Delete was unreachable until I-0058 routed both
      call sites in `Todo.tsx` through the API layer.)
- [ ] A rejected mutation records nothing.
- [ ] Read-only calls record nothing.
- [ ] A stamp written yesterday reads as not-touched today.
- [ ] Malformed stored values read as not-touched rather than throwing.
- [ ] A throwing `localStorage` accessor is caught; callers see `false`, not an exception.
- [ ] Local-day boundaries are computed in local time, not UTC.
- [ ] Tests pass.

## Blocked By

None.

## Spec Reference

FS-KSJFR §Requirements (R11–R15), §Decisions and why (D3). D3 records why this is a
localStorage stamp and not a server-side `lastActiveAt`, including what that costs.

## TDD Approach

- RED: a test asserting `wasTouchedToday()` is false after a *failed* `updateChecklistItem`,
  and true after a successful one.
- GREEN: the module, plus the call in the API layer's mutation path.
- Then: the storage-hostile cases (missing key, garbage value, throwing accessor) and the
  yesterday-stamp case, each as its own row.
