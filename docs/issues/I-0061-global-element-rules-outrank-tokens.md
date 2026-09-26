---
id: I-0061
status: in-progress
implements: FS-none
blocked_by: []
labels: [bug]
title: "Global element rules sit outside @layer and outrank Tailwind tokens, worst in dark mode"
---
Implements FS-none — predates the spec system. (`develop`'s anchor gate expects FS-NNNN or
ADR-NNNN, so this may need picking up by hand.)

## What's wrong

`client/src/app/globals.css:80-120` declares bare element rules — `p`, `h2`, `h3`, and a
`.dark body, .dark p, .dark h2, .dark h3` override — as plain CSS after `@tailwind utilities`,
outside any `@layer`. So they compete with Tailwind utilities on raw specificity and win:

- `.dark h2` (0,1,1) beats `.text-primary` (0,1,0) — a coral heading renders cream in dark mode.
- `.dark p` (0,1,1) beats `.text-muted-foreground` — every muted paragraph renders
  full-strength cream in dark and grey in light, inverting the intended hierarchy.

This contradicts `client/docs/design-guideline.md`'s central rule ("style from semantic tokens
/ CSS variables… dark mode is handled at the token layer, never with per-component overrides"):
the token layer is silently not in charge. It also has teeth — FS-KSJFR R48–R50 required a
workaround (avoiding `<p>` for muted lines) to stay conformant, because the compliant spelling
does not work. The rules also hardcode `rgb(...)` values, which is the same violation one level
down.

## What to Build

- Wrap the element rules in `@layer base`. Utilities then live in a later cascade layer and win
  by layer order regardless of specificity.
- Replace the hardcoded `rgb(...)` values with `hsl(var(--foreground))` / `hsl(var(--primary))`.
- Once the tokens redefine themselves under `.dark`, the whole
  `.dark body, .dark p, .dark h2, .dark h3` block can be deleted.
- Retire the element-avoidance workarounds in `client/src/components/home/` once utilities win.

**Blast radius is every page** currently relying on the present cascade, which is why this was
kept out of FS-KSJFR's slices. It wants its own change and a deliberate visual pass in both
themes, not a drive-by.

## Acceptance Criteria

- [x] Element rules live in `@layer base`.
- [x] A `text-primary` utility beats the global heading colour in dark mode.
- [x] A `text-muted-foreground` utility beats the global `p` colour in both themes.
- [x] No hardcoded `rgb(...)` remains in the element rules.
- [x] The `.dark` element-override block is gone.
- [x] Contrast measured against the real painted surface in dark, and the one failing class
      retokenised. Measured in-browser on the running worktree (`#1f1f1f`): `text-gray-500`
      **3.41:1 — fails AA**; `text-gray-400` 6.49:1, `text-muted-foreground` 6.10:1,
      `text-primary` 5.98:1, `text-red-400` 5.96:1, plain `p`/`h2` 11.03:1 — all pass. The
      `text-primary` utility was confirmed to win on an `h2` in dark, which is the bug this
      issue opened on.
- [x] All 24 `text-gray-500` occurrences in live surfaces replaced with `text-muted-foreground`
      (`calendar/page`, `plan/[planId]`, `myplans`, `NotesContainer`, `Todo`, `Calendar`).
      Two special cases: a muted delete button became `hover:text-destructive`, and
      `italic text-gray-400 dark:text-gray-500` lost its per-component dark override (R50).
- [x] Seen by eye in dark: the signed-out landing and `/auth`. Coral `h1`, cream body, no
      black-on-charcoal — the token refactor resolves correctly.
- [ ] **Signed-in pages not seen by eye.** `/myplans`, `/calendar`, `/plan/[planId]` and the
      notes panel were verified by measurement, not observation: the worktree client runs on
      :3011 and the gateway's CORS allows only :3010
      (`services/api-gateway/config/routes.go`), so it cannot authenticate. Every colour class
      remaining on those pages measures above AA, so this is a confirmation step, not an open
      risk. Someone signed in on :3010 with this branch checked out closes it in two minutes.

**Out of scope, found while looking:** `input.tsx` and `textarea.tsx` carry
`placeholder:text-gray-500`, visibly dim in dark on `/auth`. Placeholders were never touched by
the `.dark p` override so this is not a regression from this change — it is the shadcn drift
the design guideline already lists under *Known drift*, and it wants its own pass.

## Blocked By

None.

## Spec Reference

FS-none. Surfaced by FS-KSJFR R48–R50 conformance and the I-KSJFR-4 code review;
`client/docs/design-guideline.md` is the authority this restores.
