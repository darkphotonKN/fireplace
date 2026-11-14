---
id: I-49CQ2-3
status: done
implements: FS-49CQ2
blocked_by: [I-49CQ2-2]
labels: [feature]
title: "FS-49CQ2 slice 3: the rail's hover peek only starts on a device that can hover"
---
Implements FS-49CQ2 §Peek needs a real hover

## What to Build

On a touch tablet at ≥768px, a tap on the rail currently fires a synthetic `mouseenter`, starts
the FS-S6WVY peek, and leaves an overlay that only closes on `mouseleave`. Gate the peek on a real
hover.

- The peek (FS-S6WVY R21–R22) starts only under `(hover: hover)` — e.g. check
  `matchMedia('(hover: hover)')` when the hover-intent timer would start, or start the peek from a
  pointer event with `pointerType === 'mouse'`. Unknown/false means no peek.
- The rail toggle still pins open on tap and writes `sidebar=open` (an explicit toggle, R3-legal).
- With a mouse, the peek is exactly as before.

Blocked by slice 2 only because both edit `Sidebar.tsx` (slice 2 extracts the panel body); the
behaviour is independent.

Files expected to touch:
- `client/src/components/Sidebar.tsx` (gate `startPeek`)
- `client/src/components/Sidebar.test.tsx` (new no-hover cases; existing peek cases run with
  `(hover: hover)` true)
- `client/src/test/motion.ts` (only if the stub needs a `(hover: hover)` answer)

## Acceptance Criteria

- [ ] With `(hover: hover)` false, entering the rail does not peek; the rail toggle still pins
      open and writes `open`.
- [ ] With `(hover: hover)` true, the existing FS-S6WVY peek tests stay green unchanged in intent.
- [ ] `npm run lint`, `npm run build` and `npm test` pass.

## Blocked By

I-49CQ2-2 (same file)

## Spec Reference

FS-49CQ2 §Peek needs a real hover: R11; user stories 12–14; edge states "Touch tablet ≥768px",
"Hybrid device". Supersedes FS-S6WVY R21–R22 below md (no rail there) and gates them by hover
capability above it.

## TDD Approach

- RED: stub `(hover: hover)` false, hover the rail past the 150ms delay, assert the panel is not
  peeking (`data-peek` false, panel `inert`).
- GREEN: the hover-capability check in `startPeek`.
