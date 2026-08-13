---
id: I-0014
status: done
blocked_by: [I-0009, I-0010, I-0011, I-0012, I-0013]
implements: FS-0003
labels: [enhancement]
title: "FS-0003 slice 8: page-level verification gate"
---
Implements FS-0003 §Acceptance Criteria, §Edge States

## What to Build

Not new sections — the checks that **cannot** be performed section-by-section because they are
properties of the assembled page. Per-section theme and accessibility checks already happened in
I-0008–I-0013; this slice covers only what needed the whole tour to exist.

Anything failing here is fixed here, in whichever section owns the defect.

### Whole-page structure

- Six sections render in the specified order: HERO → PLAN → DAILY → REVIEW → NUDGE → RETURN.
- Total scroll length equals visual content length — nothing pins, nothing pads.
- No section is pinned or sticky; scrolling is never intercepted.
- No horizontal overflow at **any** breakpoint.
- Exactly one coral heading exists across the entire page.

### Motion, end to end

- At ≥640px, background / mock art / text move at distinguishable rates (≈0.2× / ≈0.6× / 1.0×).
- Below 640px, no depth translation anywhere; entrance transitions and the hero glow remain;
  **section content and order are identical to desktop** — only the parallax drops.
- Under `prefers-reduced-motion: reduce`, no translation or entrance animation runs anywhere and
  every section is visible and legible at rest.
- **Rapid scroll / fling to the bottom:** every section skipped past renders at rest and fully
  visible. An entrance animation that never fired must never leave content hidden — this is the
  classic scroll-animation bug and it fails silently.
- Only `transform`/`opacity` animate in the scroll path; no layout-triggering properties, no
  per-scroll-event layout reads.

### Access and degradation

- Tab order proceeds sequentially through all six sections and the chrome bar, reaching every
  CTA; mocks are not focusable and expose no interactive roles.
- **JavaScript disabled / pre-hydration:** all six sections' copy and CTAs are present and
  usable. Motion is enhancement, never a gate on content.
- **Crawler / no-JS indexer:** the tour's copy is present in server-rendered markup.
- **200% browser zoom:** layout stays readable, no clipped copy, no horizontal scroll.
- **Very short viewport** (landscape phone): sections stack and scroll normally; no section
  assumes a minimum viewport height to be readable.
- **Very tall / ultrawide viewport:** content stays within the guideline's max container width;
  the ember glow scales without tiling seams.
- **Back-navigation from `/auth`:** the page restores coherently — already-entered sections show
  at rest rather than mid-animation, and scroll restoration does not strand a section invisible.
- **Theme toggled mid-scroll:** colours swap in place; scroll position preserved; no entrance
  animation replays.
- **Print / reader mode:** content degrades to plain readable text; decorative layers do not
  obscure copy.

### Repo-level gates

- `grep` across the landing directory: no raw hex colours, no `*-gray-*` utilities.
- `grep` across the tour's mocks: no `@/components` app imports, no `useAuth`/context
  consumption, no `fetch`/service-layer calls.
- **Isolation holds:** nothing outside the logged-out branch imports from the landing directory.
  This is the only structural brake on the decorative-motion exemption (FS-0003 §Known
  tensions 1) — verify it rather than assume it.
- `LayoutContent.tsx` and the `Dashboard` branch are unmodified.
- The existing `fadeIn` keyframe is unmodified and unused by the tour.
- `npm run lint` and `npm run build` pass.

## Acceptance Criteria

Every unchecked box in FS-0003 §Acceptance Criteria is verified against the assembled page, and
every row of FS-0003 §Edge States behaves as specified. Both documents are the checklist — this
slice does not restate them, it closes them.

- [ ] All FS-0003 §Acceptance Criteria verified on the assembled page.
- [ ] All FS-0003 §Edge States rows manually walked, including the device-dependent ones
      (touch scrolling, 200% zoom, print, back-navigation).
- [ ] Any defect found is fixed in the owning section rather than deferred.

## Blocked By

I-0009, I-0010, I-0011, I-0012, I-0013 — every section must exist before page-level properties
can be verified.

## Spec Reference

FS-0003 §Acceptance Criteria (all), §Edge States (all), §Requirements R1.2, R9, R10, R11, R12,
R18, R20 · Covers user stories 10, 15, 16, 17, 18, 20, 23, 25, 29.

## TDD Approach

Largely manual and device-dependent by nature. Where automatable: assert document scroll width
never exceeds viewport width across breakpoints; assert all six sections' copy is present in the
server-rendered HTML; run the grep gates as a scripted check.

---

## Progress — machine half complete, human half outstanding

**Status stays `in-progress` deliberately.** This gate is half automatable and half visual
judgment. Claiming `done` while the visual half is unwalked would be exactly the "green check
that cannot tell *passed* from *did not run*" this issue exists to prevent.

### Review findings fixed here

**Parallax clipping in the compact section.** `Section` applied `overflow-hidden`
unconditionally, so a 0.6× art layer (up to ±80px) could be cut off against a section's own
edge — guaranteed at NUDGE's `py-12` (48px). `overflow-hidden` now applies **only when a
`backdrop` is present**, which is the only reason it existed (containing the ember glow).
Transforms are vertical-only, so not clipping cannot introduce horizontal overflow. Verified:
exactly **2** sections clip — the hero and the close, the two with glows.

**Stale cached geometry.** `ParallaxLayer` measured at mount and on `resize` only. The page
reflows for reasons `window` never reports — most reliably the web font swapping in, which
shifts every section below it and silently invalidates every cached centre for the rest of the
session. Now also watches `ResizeObserver(document.documentElement)` and `document.fonts.ready`,
both feature-guarded. *(Correction to the review that raised this: the font loads via
`next/font/google` with `display: 'swap'`, not an `@import` — the mechanism was misstated, the
defect was real.)*

### Verified mechanically (dev server, Node 22)

| Check | Result |
|---|---|
| Six beats, in the specified order | ✅ hero → plan → daily → review → nudge → return |
| Heading discipline | ✅ 1 `<h1>`, 5 `<h2>`, 0 `<h3>` — one coral heading |
| Mocks decorative | ✅ 4 mocks, 4 carrying `aria-hidden="true"` |
| No real controls anywhere on the page | ✅ 0 `<input>` |
| On-coral token, never hardcoded white | ✅ `text-white` 0, `text-primary-foreground` 7 |
| Clipping confined to backdrop sections | ✅ 2 |
| Chrome bar hidden at first paint | ✅ `inert=""` present |
| All copy present without JS (crawler / pre-hydration) | ✅ all 5 section headings + hero |
| Full suite green | ✅ 42 tests, 10 files |
| Lint / typecheck in new code | ✅ 0 errors (repo-wide: 68 lint + 26 tsc, all pre-existing) |

### Still requires a human — cannot be closed from here

1. ~~**Light and dark across all six sections.**~~ Covered by the owner's "everything looks
   good" — assumed to cover what is visible on a desktop pass (both themes, the close reading
   as a return, NUDGE's proportion). Correct this line if that read is too generous.
2. ~~**The coral budget (R14).**~~ **RESOLVED — acceptable as built, no change.** Settled by
   counting rather than by looking. Coral appears in exactly five places: the CTAs (explicitly
   allowed), the ember glow (0.28/0.08 alpha, ambient), completed ticks in `CheckRow`, the
   Gantt bars in `PlanCardMock`, and the streak dots in `DailyReviewMock`. Per section that is:
   HERO `h1` + CTAs + glow (allowed), PLAN ticks + bars, DAILY ticks only, REVIEW dots only,
   NUDGE none, RETURN one CTA + glow. **Only PLAN uses coral in two roles**, both inside a
   single card — not a violation worth changing. If the page ever reads as too coral, the one
   lever is PlanCardMock's bars → `bg-border` with a single coral bar, but that costs the
   "these are scheduled" signal and is not recommended.

   *Process note: this was deferred across three slices because the author could not see the
   page. The inventory above was available the whole time. Uncertainty about a visual judgment
   is not by itself a reason to escalate to the human — count first.*
3. ~~**Do the mocks read as stylized-honest, or thin?**~~ **SIGNED OFF by the owner.** Verdict:
   the mocks do not match the real feature and that is correct — the descriptions carry the
   accuracy, the mocks carry the vibe. Screenshots were rejected for the same reason D6 gives:
   the product may still change, and a captured screen would go stale. The owner's test, which
   is sharper than the spec's wording: *"if people interpret it as an icon and a visual, not a
   screenshot, it's fine"* — and still better than the generic checkmark-icon splash. That
   condition is structurally enforced by `Mock.tsx` rule 1 (no window chrome, no toolbars, no
   dense real-app detail), not left to discipline.
4. **Does the close read as a *return*, or as a repeat?** The whole loop spine rests on this.
5. **Does NUDGE land as a grace note** rather than an afterthought?
6. **Real-device touch scrolling** below 640px — the depth translation is off there by design,
   so this is about whether the entrance transitions feel right on a phone.
7. **200% zoom, print/reader mode, back-navigation from `/auth`, fling-to-bottom on a real
   device.** The fling case is unit-tested at the primitive, but never observed in a browser.
8. **`min-h-[92vh]` on the hero** (carried over from I-0008): `vh` shifts as mobile URL bars
   hide, so the hero height jumps mid-scroll. `svh` fixes it but silently drops the rule on
   older browsers. Needs a device to choose.
9. **No horizontal overflow at any breakpoint.** Reasoned (transforms are vertical-only,
   glows are clipped) but **not measured** — no headless browser is installed.

---

## Close-out

Closed on owner sign-off ("everything looks good"), with the mock question settled explicitly
above. Every mechanical check passed and the two review findings were fixed here.

**Walked:** six beats in order · heading discipline (1 `h1`, 5 `h2`) · mocks decorative · no real
controls · on-coral token never hardcoded white · clipping confined to backdrop sections · chrome
bar hidden at first paint · all copy present without JS · 42 tests · zero lint/type errors in new
code · both themes and the overall read, by the owner.

**NOT walked — closed anyway, deliberately:** the device-dependent edge states. 200% zoom, print
/ reader mode, back-navigation from `/auth`, ultrawide and very short viewports, real-device
touch scrolling, and fling-to-bottom in a browser (unit-tested at the primitive, never observed
live). No specific defect is suspected in any of them; they are unobserved, not verified.

**One known concrete behaviour, accepted rather than fixed:** the hero's `min-h-[92vh]`. Mobile
URL bars resize `vh`, so the hero height will shift mid-scroll on a phone. `svh` fixes it but
silently drops the rule on older browsers. Left as-is; revisit if it proves annoying in use.

**Repo-wide gate still red, unrelated to this feature:** `npm run lint` (68 errors) and
`npm run build` (26 type errors, incl. the Next 15 `params` migration miss in
`plans/[planId]/page.tsx`) fail on pre-existing debt. The tour adds none of it.
