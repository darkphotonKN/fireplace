# FS-0003: Product tour on the logged-out landing

> Status: work-order · SPECIFICATION.md: `client` → `## Onboarding` → `- [ ] Product tour on the logged-out landing → FS-0003` · Related ADRs: none (see §Design decisions D8)

## Summary

The logged-out landing at `/` is currently a hero, a tagline, and two buttons — a visitor
leaves knowing the product's name and nothing about what it does. This feature replaces it
with a scroll-told product tour whose six sections walk the visitor through Fireplace's own
working loop (plan → daily → review → nudge → back to the hearth), each illustrated by a
mocked visual. Everything shown is static and fabricated: no API calls, no real data, no
interactivity beyond navigation. Restrained parallax gives the scroll depth, and the whole
page is built from the existing warm design tokens so it reads as the hearth the product
claims to be.

**Visual authority:** `client/docs/design-guideline.md` wins over this document on any
appearance question.

## Design decisions (from scoping)

| # | Decision | Rationale |
|---|---|---|
| D1 | "Useful" = show-don't-tell, not do-something | Keeps the feature purely frontend and literally "just visual", while still fixing the real complaint — a visitor learns nothing today |
| D2 | Logged-out branch only; `Dashboard` untouched | Smallest blast radius, and the divergence is correct: a landing is a marketing surface, a dashboard is a tool |
| D3 | Spine = the product's own loop, closing where it opened | The scroll *is* the loop; gives parallax something to mean rather than decorate |
| D4 | Restrained depth (3 layers), no pinning | Text never moves relative to the reader, so legibility is never traded for effect |
| D5 | Chrome-free hero; slim bar fades in after it | Preserves the uninterrupted first impression while keeping the CTA one click away in sections 2–6 |
| D6 | Mocks are stylized-honest, not replicas | A near-replica silently becomes a lie the first time the real UI changes; hand-built mocks cannot track the app |
| D7 | Mocks are bespoke presentational components | Importing real components drags in `AuthContext`, the service layer, and client state, coupling marketing to product internals |
| D8 | The principle-5 exemption is **not** recorded as an ADR | Offered during scoping and declined. Consequence accepted: the rationale governs this feature only — see §Known tensions |

## Known tensions

Both are with `client/docs/design-guideline.md`, which is the visual authority. Recorded so a
reviewer meets them as decisions rather than as violations.

1. **Principle 5 — "Motion communicates, never decorates."** Parallax is decorative by
   definition. The position taken: a marketing surface is a different rhetorical register than
   task UI, so the landing may bend principle 5 while product surfaces stay bound by it. This
   boundary is *not* recorded repo-wide (D8), so a reviewer may still reasonably raise it, and
   nothing formally prevents the exemption being cited as precedent for animating product
   surfaces. Confining it to the landing is this spec's job (R1.2).
2. **Principle 3 — "One coral moment per view."** Six sections each wanting a coral heading
   would shred it. Resolved by R14.

## Starting state

Facts verified against the code at spec time; the implementer should not re-derive them.

- `client/src/app/page.tsx` holds two components behind one route. `Home` branches on
  `useAuth()` into `LandingPage` (logged out, ~35 lines) and `Dashboard` (logged in).
- `LandingPage` hardcodes raw hex — `rgb(247, 111, 83)`, `#2e2e2e`, `#d1cfc0` — instead of
  semantic tokens, violating design-guideline principle 2.
- `LayoutContent.tsx:83` short-circuits for logged-out `/`
  (`if (isHomePage && !isAuthenticated) return <>{children}</>`), so **the landing renders
  with no chrome at all** — no header, no logo, no theme toggle. The tour must supply its own.
- `globals.css` styles headings with **global element selectors carrying raw hex**:
  `h1 { color: rgb(247,111,83); font-size: 32px; font-weight: 700 }`, plus `h2`/`h3`/`p` and a
  `.dark body, .dark p, .dark h2, .dark h3 { color: rgb(209,207,192) }` override block.
  Consequences: a bare `<h1>` is already coral and already clamped to 32px, so the hero needs
  an explicit size utility to reach the guideline's hero scale (`text-4xl`+ = 38px+).
- **`--primary-foreground` is undefined** in both `:root` and `.dark`, but
  `tailwind.config.js:48` consumes `hsl(var(--primary-foreground))` — so
  `text-primary-foreground` resolves to nothing. This is why today's landing hardcodes
  `text-white` on the coral button. The design guideline already flags it as an open "token
  gap to close" and prescribes the fix.
- **No `prefers-reduced-motion` handling exists** anywhere in `globals.css`.
- An existing `fadeIn` keyframe (600ms, `translateY(-8px)`, coral halo pulse) is reserved for
  *newly added checklist items landing in a list* — a communicating motion. The tour's
  entrance is a different intent and must not overload it.
- `ThemeToggle` (`src/components/ThemeToggle.tsx`) and `Logo` (`src/components/Logo.tsx`,
  wrapping `FireIcon`) already exist and are reusable in the tour's chrome bar.
- `client/CONTEXT.md` is empty scaffolding — no glossary constraints on this feature's prose.

## Requirements

### Structure

**R1.1** — The tour renders only for unauthenticated visitors at `/`. The `Dashboard` branch
is not modified.
**R1.2** — All tour code (sections, mocks, motion) lives under a dedicated landing directory
and is imported by nothing outside the logged-out branch, so the principle-5 exemption cannot
leak into product surfaces.

**R2** — The tour has exactly **six** sections in this order, telling the product's loop:

| # | Section | Beat | Mocked visual |
|---|---|---|---|
| 1 | HERO | "Sit down by the fire." | ambient ember glow |
| 2 | PLAN | "Every arc gets a plan." | plan card + soft Gantt bars |
| 3 | DAILY | "Today is a short list." | checklist, daily vs long-term |
| 4 | REVIEW | "Close the day honestly." | daily review + streak |
| 5 | NUDGE | "It notices what you keep skipping." | single AI suggestion card |
| 6 | RETURN | back to the hearth | — |

**R2.1** — Section 5 (NUDGE) is deliberately the smallest section — one suggestion card, less
vertical space — so it reads as a grace note rather than padding.
**R2.2** — Section 6 visually returns to section 1's imagery (the hearth), making the cycle
legible rather than merely stated.

**R3 — Hero.** Carries the anchor phrase **verbatim**: *"Start your plan now. Sit down by the
fire."* It is documented product voice; rewording it is a brand decision wider than this page.
The hero renders chrome-free. Its `h1` requires an explicit size utility to exceed the
globally-set 32px.
**R3.1** — The existing one-sentence description ("A developer and learning platform that
houses your checklists…") is replaced with a sub-line that front-loads meaning rather than
jargon and hints that the page continues below.

**R4 — Chrome bar.** Absent over the hero; fades in once the visitor scrolls past it. Carries
logo, theme toggle, and a primary CTA. It is the tour's own component — nothing is inherited
from `LayoutContent`.

**R5 — Closing CTA.** Section 6 presents **one** primary action (`Start your plan`) plus a
quiet `Sign in` text link. Not an equal-weight pair — by section 6 the visitor is warm, and
two equal buttons read as hesitation.

**R6 — CTA targets are unchanged:** `/auth` and `/auth?tab=signup`. The auth page itself is
out of scope.

### Motion

**R7 — Depth layers** (viewports ≥ `sm`): background ember glow/grain ≈ **0.2×** scroll speed;
section mock art ≈ **0.6×**; headings and body text **1.0×** (native).
**R8 — Entrance:** each section fades and rises on enter — `opacity 0→1`, `y +16px→0`, ≈400ms
ease-out — via a **new keyframe**, not the existing `fadeIn`.
**R9 — Prohibited:** pinned/sticky sections, scroll hijacking, horizontal scroll, and any
scroll-driven mutation of content (no self-ticking checkboxes, no self-drawing bars). Page
scroll length must equal visual length.
**R10 — Below `sm` (640px):** depth translation is disabled entirely; entrance transitions and
the ambient hero glow remain. Section content and order are identical to desktop — only the
parallax drops.
**R11 — `prefers-reduced-motion: reduce`:** all parallax translation and all entrance animation
collapse to a static page. The tour must be fully comprehensible with zero motion — no content
may depend on animation to become visible or legible.
**R12 — Performance:** the scroll path animates `transform` and `opacity` only, driven by
`requestAnimationFrame` or CSS; no layout-triggering properties, no per-scroll-event layout
reads.

### Visual

**R13 — Token discipline.** All colour comes from semantic tokens (`bg-background`,
`text-foreground`, `text-muted-foreground`, `bg-primary`, `border-border`). No raw hex, no
`*-gray-*` utilities. Dark mode is handled at the token layer only.
**R14 — Coral discipline.** The hero `h1` is the page's coral moment. Section headings take
`foreground`. Coral is otherwise used only for CTAs and at most one small accent per section.
**R15 — `--primary-foreground` is defined** in `globals.css` for both `:root` and `.dark` — the
on-coral text colour, cream `#F2F0E3` (light) / near-white (dark) — so `text-primary-foreground`
resolves. Coral CTAs use it; no CTA hardcodes `text-white`.
**R16 — Mocks are stylized-honest:** same tokens, type scale, radii and coral as the product,
but simplified — a checklist is a few rows, a Gantt is a few soft bars. No window chrome, no
toolbars, no dense real-app detail.
**R17 — Mocks are bespoke presentational components** containing no imports from real app
components, no context consumption, no service-layer calls, and no fetches.
**R18 — Both themes.** Every section, every mock, and the ember glow are verified in light and
dark. Warm neutrals only: no cold grays, no pure `#FFFFFF`, no pure black.

### Behavior & access

**R19 — First paint.** The hero paints immediately without waiting on auth resolution; the
`"Loading..."` string no longer gates the landing. An authenticated visitor arriving at `/`
may briefly see the hero before the dashboard replaces it — accepted (see Edge States).
**R20 — Accessibility.** Focus order stays sequential through all six sections and the fade-in
chrome bar. Mocks are decorative: not focusable, not announced as interactive controls, and
carrying no misleading roles or labels. Real CTAs remain reachable and operable by keyboard.
**R21 — Stack conventions** from `client/CLAUDE.md` bind: Tailwind v3, `cn()` for conditional
classes, `next/font` / `next/image` / `next/link`, no `any`, and the `"use client"` boundary
pushed as far down as possible — motion is a client island, not a whole-page directive.

## User Stories

1. As a first-time visitor, I want to see what Fireplace actually does before deciding, so that
   I'm not asked to sign up for something I can't picture.
2. As a first-time visitor, I want the page to look like the product feels — warm and unhurried
   — so that I can judge whether it suits how I work.
3. As a first-time visitor, I want to see a plan with real-looking tasks and dates, so that I
   understand plans are more than a to-do list.
4. As a first-time visitor, I want to see the daily list separated from long-term goals, so
   that I grasp the dual-scope idea without reading documentation.
5. As a first-time visitor, I want to see what closing out a day looks like, so that I know the
   product expects reflection, not just checking boxes.
6. As a first-time visitor, I want to see one example of the AI noticing something, so that I
   know suggestions are grounded rather than generic.
7. As a first-time visitor, I want the page to end where it began, so that the loop registers
   as a cycle I'd be joining rather than a list of features.
8. As a visitor scrolling, I want depth and gentle movement, so that the page feels crafted
   rather than templated.
9. As a visitor scrolling, I want the text to hold still while I read it, so that the effect
   never costs me legibility.
10. As a visitor scrolling, I want the page to end when it looks like it should, so that I'm
    never trapped in a section that won't release the scroll.
11. As a visitor who has read a few sections, I want a way to sign up without scrolling back,
    so that acting on interest is immediate.
12. As a visitor, I want the first thing I see to be the product, not a "Loading…" string, so
    that my first impression isn't of an unfinished app.
13. As a visitor who prefers dark mode, I want the tour to be equally warm and legible in dark,
    so that the theme isn't an afterthought.
14. As a visitor, I want to switch theme from the landing itself, so that I can see it in the
    mode I'd actually use.
15. As a visitor on a phone, I want the tour to scroll smoothly, so that my first impression
    isn't jank.
16. As a visitor on a phone, I want the same six beats in the same order as desktop, so that I
    get the whole story on the device I happen to be holding.
17. As a visitor with `prefers-reduced-motion` set, I want the tour to be completely static and
    still make sense, so that my accessibility setting doesn't cost me content.
18. As a keyboard-only visitor, I want to tab through the page in a sensible order and reach
    every CTA, so that I can sign up without a mouse.
19. As a screen-reader user, I want decorative mocks to stay silent, so that the narration is
    the copy, not a recital of fake checkbox labels.
20. As a visitor who zooms to 200%, I want the layout to stay usable, so that the design doesn't
    assume perfect vision.
21. As a returning logged-out visitor, I want the CTAs to land me on the same auth page as
    before, so that nothing I already know has moved.
22. As an authenticated user who types the bare domain, I want to end up at my dashboard, so
    that the marketing page doesn't get between me and my work.
23. As a visitor who hits back from the auth page, I want to return to a coherent page state, so
    that the tour doesn't reset or land me mid-animation.
24. As a visitor on a slow connection, I want the hero readable before everything below has
    loaded, so that the page is useful immediately.
25. As a search crawler, I want the tour's copy present in the markup, so that the landing is
    indexable on what the product does.
26. As a designer, I want the tour built from the existing tokens, so that a palette change
    updates the landing for free.
27. As a designer, I want the mocks simplified rather than pixel-faithful, so that they don't
    become stale screenshots the moment the real UI moves.
28. As a maintainer, I want the tour's components isolated from app components, so that
    refactoring the real checklist can't break the landing.
29. As a maintainer, I want the decorative-motion exemption confined to the landing directory,
    so that it can't be cited as precedent for animating the dashboard.
30. As a reviewer, I want the coral discipline explicit, so that six warm sections don't quietly
    become six competing accents.

## Acceptance Criteria

- [ ] Six sections render in the specified order; NUDGE occupies visibly less vertical space
      than REVIEW; section 6 reprises the hero's imagery.
- [ ] The hero contains the anchor phrase verbatim: "Start your plan now. Sit down by the fire."
- [ ] The hero `h1` renders at hero scale (≥38px), overriding the global 32px element style.
- [ ] No chrome bar is visible at scroll position 0; it appears after the hero is scrolled past
      and carries logo, theme toggle, and a primary CTA.
- [ ] Section 6 shows exactly one primary button plus a text-link `Sign in`.
- [ ] CTAs resolve to `/auth` and `/auth?tab=signup`.
- [ ] At ≥640px, background, mock art, and text move at distinguishable rates on scroll
      (≈0.2× / ≈0.6× / 1.0×).
- [ ] Each section fades and rises on entry via a new keyframe; the existing `fadeIn` keyframe
      is unmodified and unused by the tour.
- [ ] No section pins; scrolling is never intercepted; the document has no horizontal overflow
      at any breakpoint; total scroll length matches visual content length.
- [ ] No content mutates as a function of scroll position.
- [ ] Below 640px, no depth translation occurs; entrance transitions and the hero glow remain;
      section content and order are byte-identical to desktop.
- [ ] Under `prefers-reduced-motion: reduce`, no translation or entrance animation runs and
      every section's content is visible and legible at rest.
- [ ] Only `transform` and `opacity` are animated in the scroll path; no layout-triggering
      properties are written per scroll event.
- [ ] `--primary-foreground` is defined in `:root` and `.dark`; `text-primary-foreground`
      resolves to a visible colour; no CTA in the tour hardcodes `text-white`.
- [ ] `grep` over the tour directory finds no raw hex colours and no `*-gray-*` utilities.
- [ ] Exactly one coral heading exists on the page (the hero `h1`); section headings render at
      `foreground`.
- [ ] `grep` over the tour's mock components finds no imports from `@/components` app
      components, no `useAuth`/context consumption, and no `fetch`/service-layer calls.
- [ ] All six sections and every mock are verified in light and dark mode.
- [ ] The hero is present in first paint without waiting on auth resolution; the `"Loading..."`
      string no longer precedes the landing.
- [ ] Tab order proceeds sequentially through the page and reaches every CTA; mocks are not
      focusable and expose no interactive roles.
- [ ] The tour's copy is present in server-rendered markup.
- [ ] `npm run lint` and `npm run build` both pass.

## Edge States

| Condition | Expected |
|---|---|
| **Auth still resolving** | Hero paints immediately; nothing waits on `isLoading`. |
| **Authenticated visitor at `/`** | Dashboard replaces the tour once auth resolves. A brief hero flash is accepted (R19) — the alternative was gating first paint, which was rejected. |
| **`prefers-reduced-motion: reduce`** | Fully static page, all content at rest and legible. |
| **Touch device / iOS momentum scroll** | Below `sm`, no scroll-linked transforms run at all, so momentum scrolling cannot desynchronise from layer positions. |
| **JavaScript disabled or not yet hydrated** | All six sections' copy and CTAs are present and usable; only the motion is absent. Motion is enhancement, never a gate on content. |
| **Slow network** | Hero is readable before below-fold mock art has loaded; nothing below shifts the hero when it arrives. |
| **Very short viewport** (landscape phone) | Sections stack and scroll normally; no section assumes a minimum viewport height to be readable. |
| **Very tall / ultrawide viewport** | Content stays within the guideline's max container width; the ember glow scales without tiling seams. |
| **200% browser zoom** | Layout stays single-column-safe and readable; no clipped copy, no horizontal scroll. |
| **Theme toggled mid-scroll** | Colours swap in place; scroll position and any in-flight entrance animations are preserved — no re-entry animation replay. |
| **Back-navigation from `/auth`** | Page restores to a coherent state; already-entered sections are shown at rest rather than mid-animation, and scroll position restoration does not strand a section invisible. |
| **Rapid scroll / fling to bottom** | Sections skipped past render at rest, fully visible — an entrance animation that never fired must never leave content hidden. |
| **Crawler / no-JS indexer** | Copy is in the markup and indexable. |
| **Print / reader mode** | Content degrades to plain readable text; decorative layers do not obscure copy. |

## Out of Scope

- Any interactivity beyond navigation — no interactive demo, no local checklist, no draft
  carried into signup.
- The logged-in `Dashboard` branch and its visual alignment with the new landing.
- The `/auth` page.
- Any backend work. This feature adds and changes **no HTTP endpoints**, so it carries no
  `§API surface` section and ADR-0002 serialize-on-touch does not apply.
- Rewording the anchor phrase or otherwise reopening documented product voice.
- Fixing `globals.css`'s global element selectors and their raw hex values wholesale. The tour
  works within them and adds `--primary-foreground` (R15) plus reduced-motion handling (R11);
  the broader token cleanup is separate.
- Converting the rest of `client/SPECIFICATION.md` out of monolith-era prose — that file
  carried **zero** thin lines before FS-0003; `## Onboarding` is the first. Reverse-engineering
  the remainder is `/spec-bootstrap client`.

### Alternatives rejected during scoping

Recorded so they are not relitigated. The first two are **parked, not dead** — they are the
natural phase 2 if the tour converts.

| Alternative | Why rejected |
|---|---|
| Try-before-signup (interactive focus + checklist, draft carried into signup) | Stops being "just visual" — needs draft-carry plumbing and an answer for abandoned drafts |
| Narrative + one interactive hook (dashboard's focus input lifted onto the landing) | Same reason, smaller scope |
| "A day in the life" spine (dawn→dusk light shift) | More cinematic but less feature-explicit; wouldn't fix "a visitor learns nothing." The time-of-day light shift is worth stealing as an accent within the loop spine |
| "Capabilities, warmly" (hero + 3-up feature row + mock + CTA) | Fastest to build, but closest to "every other splash page" — the exact complaint |
| Cinematic motion (pinned sections, self-ticking checklists, self-drawing Gantt bars) | Heavy, and scroll length stops matching content length |
| Whisper-only motion (no true parallax) | The deliberate minimum; offered and declined — parallax was explicitly requested |
| Abstract-warmth mocks (shapes and embers, no product UI) | Beautiful and cheap, but stops *showing* the product, contradicting D1 |
| Near-replica mocks | Silently becomes a lie the first time the real UI changes |
| Redesigning the dashboard in the same pass | Rejected: landing is marketing, dashboard is a tool; divergence is correct |
