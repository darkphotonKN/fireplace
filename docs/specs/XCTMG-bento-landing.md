# FS-XCTMG: Bento landing

> Status: work-order · SPECIFICATION.md: `client` → `## Onboarding` → `- [ ] Feature showcase on the logged-out landing → FS-XCTMG` · Supersedes in part: FS-0003 (see §Relationship to FS-0003) · Related ADRs: none

## Summary

The logged-out landing at `/` keeps its one job, showing a visitor what Fireplace does before
asking them to sign up, but changes shape. Today it is FS-0003's six-section scroll tour. This
feature turns it into an x.ai-style homepage: a much larger, cleaner hero carrying the same anchor
phrase, then a **bento grid** of five feature tiles (Plan, Today, Review, Milestones, Calendar).
Each tile holds a hand-coded mock of the real screen and, on hover or keyboard focus, plays a
short animation that "shows more": a box ticks, a bar fills, the next reply slides in. Every tile
is a link to sign-up. The closing "Tomorrow, the same fire." section stays, larger. It stays
cozy: Fireplace tokens, Lora, the coral brand, and the warm dark theme.

Everything shown is still static and invented. No API calls, no real data, nothing interactive
beyond navigation and the hover reveal.

**Visual authority:** `client/docs/design-guideline.md` wins over this document on any appearance
question, except where §Known tensions names a deliberate departure.

Client terms used here (`Day plan`, `entry`, `milestone`, `step`, `progress strip`) are defined in
`client/CONTEXT.md`. "Tile", "bento grid", "end state" and "reveal" are landing layout words only
and are not domain terms.

## Design decisions

All were set as **charter defaults** under an approved `/charter` run and locked by the owner
through the charter's lock preview. None was interviewed in this session.

| # | Decision | Source | Rationale |
|---|---|---|---|
| D1 | **Bounds.** Changes are confined to `client/src/components/landing/**`, the logged-out branch of `client/src/app/page.tsx`, landing tests (including `client/src/test/mockContract.ts`), this FS, and one SPECIFICATION line. No backend, no data fetch, no product component imports in mocks, no edits to the 21 files of the x.ai light-theme pass (commit 67cb739, including `globals.css` and `tailwind.config.js`). Landing-only keyframes and hover rules go in a landing-local CSS module. | charter default | Keeps the marketing surface's blast radius inside the sealed directory, and leaves the theme pass to its own owner. |
| D2 | **Pictures are hand-coded mocks**, not screenshots, styled to look like the production screens. Crisp in both themes, animatable, no personal data. Extends FS-0003 D6/D7. | charter default | Screenshots blur at scale, go stale, can't follow the theme and can't animate a real tick or bar. |
| D3 | **One persona, one plan:** "Chris Anderson", plan "Learn microservices", with the milestone cards "Database" 1/5 and "Microservices" 6/8 (contents in R14). No long personal notes. | charter default | Continuity with `mocks/content.ts`; recognisable, real-looking, ages well. One story across the tiles reads as grounded rather than as five unrelated screenshots. |
| D4 | **Hero: same content at x.ai scale.** Eyebrow logo + wordmark, a pill badge, the two-line anchor phrase as a huge headline with a coral underline on "fire", the existing description as subhead, two pill CTAs. Chrome-free. | charter default | The anchor phrase is documented product voice; the x.ai look is scale and restraint, not new copy. |
| D5 | **Five tiles replace PLAN/DAILY/REVIEW/NUDGE**; RETURN stays as the closing CTA, enlarged. Layout per R10. | charter default | The grid shows more of the product in less scroll; the close still returns to the hearth. |
| D6 | **Hover contract:** transform/opacity only; hover and `:focus-visible` both trigger; no-hover devices play the reveal once on scroll-into-view; `prefers-reduced-motion` shows the end state statically. | charter default | Keyboard and touch visitors get the same "show more"; reduced-motion visitors get the content without the motion. |
| D7 | **Whole tile is a link to sign-up**, labelled "Try it →". | charter default | No public feature pages exist to link to. |
| D8 | **Chrome bar:** hidden over the hero, fades in on scroll; right side is theme toggle + "Sign in" (grey pill) + "Start your plan" (solid pill), x.ai-sized. No profile menu. | charter default | Logged-in users never see the landing, so a profile menu would be dead weight. |
| D9 | **Theme:** light uses the x.ai tokens (white page, ivory tiles, hairlines); dark keeps the current warm dark. CTAs use `--action`. Lora stays. | charter default | The landing should match the light re-theme the app is moving to, without forking dark. |
| D10 | **Responsive:** 1 column on phones, 2 at `md`, full bento at `lg`. No horizontal scroll. | charter default | The bento only reads at width; narrower, tiles stack in story order. |
| D11 | **Verification** in the final slice: dev server on :3011, screenshots in light/dark at desktop/mobile. | charter default | Hover and theme fidelity can't be asserted in jsdom. |
| D12 | **No [HUMAN] or owner-gated slices.** | charter default | This is delegable frontend work with no learning or judgement gate. |

Derived while writing (follow from the decisions above; flagged so a reviewer can contest them):

- **H1 colour.** The headline renders at `foreground`, not the global coral `h1`, so the coral
  underline on "fire" can be the page's one coral moment (R4). A coral underline under coral
  text would not read.
- **Parallax scope.** FS-0003's 0.6× art layer belonged to the superseded split sections. Tiles
  are hover targets and do not drift; the 0.2× ember glow behind the hero and the close survives
  (R21).
- **`md` grid.** Two columns in DOM order; the last tile (Calendar) spans both. Visual order
  always equals DOM order (R10).
- **Database's fifth item** is "Indexes and query plans" (D3 left it as "one sensible item").
- **Control radius** comes from `rounded-control`: a pill in light, the dark theme's
  existing corner in dark (D9 keeps dark as it is).

## Relationship to FS-0003

FS-0003 is shipped and write-once. It is not edited; this table is the supersession record.

| FS-0003 | Fate here | Notes |
|---|---|---|
| R1.1 logged-out branch only, `Dashboard` untouched | **survives** | |
| R1.2 sealed landing directory | **survives** | Strengthened by R2 (the CSS module lives inside it). |
| R2 six sections in loop order, R2.1 NUDGE smallest | **superseded** | By R9–R10: hero, five tiles, close. |
| R2.2 close reprises the hero's imagery | **survives** | |
| R3 anchor phrase verbatim, chrome-free hero, explicit `h1` size | **survives** | Hero *sizing* is superseded by R4. |
| R3.1 sub-line | **survives** | The current description is kept verbatim as the subhead. |
| R4 chrome-bar contents (logo, toggle, one CTA) | **superseded** | By R7. Its hidden-over-hero and fade-in behaviour survives. |
| R5 one primary + quiet `Sign in` at the close | **survives** | Enlarged, and the primary uses `--action` (R8). |
| R6 CTA targets `/auth`, `/auth?tab=signup` | **survives** | Tiles also target `/auth?tab=signup`. |
| R7 depth layers | **partly superseded** | Glow layer (0.2×) survives; the art layer (0.6×) leaves with the split sections (R21). |
| R8 entrance via `riseIn`, not `fadeIn` | **survives** | Applies to the hero copy, the grid and the close. |
| R9 no pinning, no scroll hijack, no scroll-driven content mutation | **restated** | By R19: still no mutation as a *function of scroll position*. A reveal that a hover, a focus, or one scroll-into-view *triggers* is allowed. |
| R10 below `sm` no depth translation | **survives** | |
| R11 reduced motion collapses to static | **survives** | Restated for tiles by R17 (static **end** state). |
| R12 transform/opacity only on the scroll path | **survives** | Extended to every hover reveal (R16). |
| R13 token-only colour | **survives** | |
| R14 coral discipline | **restated** | By R4/R22: the coral moment is the underline on "fire". |
| R15 `--primary-foreground` defined | **survives** | Already shipped; untouched. |
| R16 mocks simplified, no dense real-app detail | **superseded** | By R13: mocks are styled like the production screens. |
| R17 mocks bespoke, no app imports, no context, no fetch | **survives** | |
| R18 both themes verified | **survives** | Its "no pure `#FFFFFF`" clause is superseded for the light page by the x.ai tokens (§Known tensions 2). |
| R19 first paint not gated on auth | **survives** | |
| R20 sequential focus, decorative mocks silent | **survives** | Restated by R23: tiles are now focusable links; mocks inside stay `aria-hidden`. |
| R21 stack conventions | **survives** | |

## Known tensions

All with `client/docs/design-guideline.md`, recorded so a reviewer meets them as decisions.

1. **Principle 5, "Motion communicates, never decorates."** FS-0003 already bent this for the
   landing. The hover reveals here do communicate (they show what the feature does next), but
   they are also there to delight. The exemption stays confined to the sealed landing directory
   (R2).
2. **Principle 1, "no pure `#FFFFFF`".** The x.ai light pass (67cb739) set the light page to
   white. The landing consumes tokens and inherits that; it does not introduce any raw white.
   Reconciling the guideline is the theme pass's job, not this FS's.
3. **"`h1` is always coral".** The hero headline renders at `foreground` with a coral underline on
   "fire" (§Design decisions, derived). The coral moment survives; it moves from the glyphs to the
   underline.
4. **"Typography: Merriweather".** The guideline still names Merriweather; the app ships Lora via
   `next/font` in `client/src/app/layout.tsx`. The landing inherits Lora and sets no font.
5. **Glossary vs. badge copy.** The badge reads "New · Day plan with AI moves ↗". `client/CONTEXT.md`
   marks **move** superseded by **entry**. The badge string is locked marketing copy (D4) and
   is used verbatim; spec prose, code identifiers and test names say "entry".

## Starting state

Verified against the code at spec time (HEAD 67cb739).

- `client/src/app/page.tsx`: the logged-out branch is `return <LandingTour />`, not gated on
  `isLoading`. Expect no change there beyond what the grid needs (likely none).
- `client/src/components/landing/LandingTour.tsx` renders `ChromeBar`, then `HeroSection`,
  `PlanSection`, `DailySection`, `ReviewSection`, `NudgeSection`, `ReturnSection`.
- `ChromeBar.tsx`: fixed bar, `useScrolledPast(0.8)`, `inert` while hidden; logo + wordmark left,
  `ThemeToggle` + one `CtaLink` right. Imports `@/components/Logo` and `@/components/ThemeToggle`
  (allowed: chrome, not mocks; neither file is edited).
- `CtaLink.tsx`: variants `primary` (`bg-primary`), `outline`, `quiet`; exports `SIGN_UP_HREF`
  (`/auth?tab=signup`) and `SIGN_IN_HREF` (`/auth`).
- `HeroSection.tsx`: `h1` "Fireplace" at `text-5xl md:text-6xl` (coral by the global rule); the
  anchor phrase as a `p`; the description "The whole shape of a project, and the four things
  you're doing about it today. When the day closes, you write down how it actually went."; the
  two CTAs; a "See how a day here runs ↓" hint. Backdrop `EmberGlow`.
- `Section.tsx` wraps content in `RevealOnEnter` (state machine idle → hidden → revealed,
  `animate-riseIn`, `inert` while hidden, fling-past safe, reduced-motion safe).
- `EmberGlow` uses `ParallaxLayer` at `GLOW_SPEED` (0.2). `SplitSection` uses `ParallaxLayer` at
  `ART_SPEED` (0.6) and is used only by the four sections being removed.
- `motion/useMediaQuery.ts` exports `useMediaQuery`, `usePrefersReducedMotion`,
  `REDUCED_MOTION_QUERY`, `WIDE_VIEWPORT_QUERY`. There is no hover-capability query yet.
- `mocks/Mock.tsx` is the mock shell (`data-landing-mock`, `aria-hidden`, `pointer-events-none`,
  `bg-card`). Its doc comment states the "stylized-honest, never replicas" rule that R13
  supersedes. `mocks/content.ts` holds `PLAN_TITLE = 'Learn microservices'` and `GATEWAY_TASK`.
- `client/src/test/mockContract.ts` (`expectMockContract`) asserts every `[data-landing-mock]` is
  `aria-hidden` and contains no `input, button, a, [tabindex], [role]`. Reusable as is.
- Tokens available from the x.ai pass (read only): `bg-action` / `text-action-foreground`,
  `rounded-control` (pill in light, `radius - 2px` in dark), `rounded-tile` (16px light, 8px
  dark), `shadow-tile` (none in light), ivory `--card` (`#f9f8f6`) on a white light page,
  `--secondary` jet (`#0a0a0a`) with white `--secondary-foreground` in both themes, `bg-muted`
  (`#f2f1ef` light). Font scale reaches `text-7xl` (74px) and `text-8xl` (98px).
- The global `h1` rule paints coral at 32px; any `h1` needs explicit size and colour utilities.
- No CSS module exists in the landing directory yet. Tailwind v3 `hover:` is not gated by
  `(hover: hover)`, so the hover contract lives in the CSS module, not in `hover:` utilities.
- Tests are Vitest + Testing Library under jsdom (`npm test`). Dev script uses port 3010; the
  worktree runs on :3011.

## Requirements

### Structure

**R1** — The landing renders only for unauthenticated visitors at `/`; the signed-in branch is not
modified.

**R2** — All new landing code, including the CSS module that holds the tile keyframes and hover
rules, lives under `client/src/components/landing/` and is imported by nothing outside the
logged-out branch. No file of the x.ai pass is edited (D1).

**R3** — Page order: chrome bar (fixed), hero, bento grid, close. Nothing else.

### Hero

**R4** — The hero, chrome-free, top to bottom:
1. **Eyebrow:** the logo and the "Fireplace" wordmark, small (≈ `text-sm`/`text-base`). Not a
   heading.
2. **Badge:** a pill reading "New · Day plan with AI moves ↗", verbatim. An in-page link to the
   Today tile (`#today` or equivalent); it scrolls there and is keyboard operable.
3. **Headline:** the single `h1`, the anchor phrase verbatim on two lines, "Start your plan now."
   / "Sit down by the fire.", at `text-5xl` or larger on phones and `text-7xl`–`text-8xl` at
   `lg`, colour `foreground`. The word "fire" carries a coral (`primary`) underline. The
   underline is the page's single coral moment.
4. **Subhead:** the existing description, verbatim, at ≈ `text-xl`–`text-2xl`, `muted-foreground`.
5. **CTAs:** two pills: "Start your plan" (solid, `bg-action text-action-foreground`) to
   `/auth?tab=signup`, and "Sign in" (grey, `bg-muted text-foreground`) to `/auth`.
6. The scroll hint "See how a day here runs ↓" may stay, quiet.

The spoken headline keeps a pause between its two halves (the existing `{' '}` before the break).

**R5** — The hero fills about one viewport on desktop and never needs horizontal scroll; at 320px
wide the headline wraps rather than overflowing.

**R6** — The `EmberGlow` backdrop stays behind the hero.

### Chrome bar

**R7** — The chrome bar stays absent over the hero and fades in once the visitor scrolls past it
(existing `useScrolledPast` behaviour, `inert` while hidden). Left: logo + wordmark. Right, in
order: theme toggle, "Sign in" (grey pill, `/auth`), "Start your plan" (solid `--action` pill,
`/auth?tab=signup`), sized like x.ai's (≈ `text-sm`, `px-4 py-2`). No profile menu.

### CTAs

**R8** — `CtaLink` gains the two pill variants the hero and bar use: a solid action pill
(`bg-action text-action-foreground rounded-control`) and a grey pill (`bg-muted text-foreground
rounded-control`). Every solid CTA on the page uses `--action`, including the close. No CTA
hardcodes a colour.

### Bento grid

**R9** — Five tiles, in this DOM order: **PLAN**, **TODAY**, **REVIEW**, **MILESTONES**,
**CALENDAR**. They replace `PlanSection`, `DailySection`, `ReviewSection` and `NudgeSection`.

**R10** — Layout:
- below `md`: one column, DOM order;
- `md`: two columns in DOM order; CALENDAR (the fifth tile) spans both columns;
- `lg`: row 1 is PLAN, TODAY, REVIEW as three equal tiles; row 2 is MILESTONES spanning two
  columns and CALENDAR one.

Visual order always equals DOM order (no `grid-flow-dense`, no `order-*`), so tab order matches
what the eye sees. The grid stays within the page's max container width. No horizontal overflow
at any width.

**R11** — Each tile is one shell (a shared `BentoTile` component) that is:
- a single `next/link` to `/auth?tab=signup` covering the whole tile;
- `rounded-tile`, `bg-card` with a `border-border` hairline, `shadow-tile`;
- laid out with the mock on top, then a footer row with the tile's **label** bottom-left (PLAN,
  TODAY, REVIEW, MILESTONES, CALENDAR, or a short title-case line naming the feature) and
  "Try it →" bottom-right;
- fixed in size between start and end state: the reveal never changes the tile's box or moves
  anything outside it.

**R12** — TODAY is the one inverted tile: `bg-secondary text-secondary-foreground` in both themes,
with its inner colours drawn from `secondary-foreground` at reduced opacity rather than from the
page tokens, which would not read on it. Its footer row follows the same inversion.

### Mocks and content

**R13** — Mocks are hand-coded presentational components styled to look like the production
screens they stand for: the Day plan section's groups and day meter, the milestone cards, the
progress strip, the calendar card. Same tokens, Lora, radii and density as production. They stay
bespoke: no imports from `@/components` product components, no context, no service layer, no
fetch (FS-0003 R17). They stay decorative: wrapped in `Mock` (`data-landing-mock`,
`aria-hidden`), no real inputs, no focusable or role-bearing nodes. `Mock`'s doc comment is
updated to say "styled like production" instead of "stylized-honest".

**R14** — All invented content lives in `mocks/content.ts`, one story:
- persona **Chris Anderson**; plan **Learn microservices**;
- milestone **Database**, 1/5: SQL Tools Drill Basics ✓, SQL Tools Drill Practice, Advanced SQL
  Queries Practice, Migration strategies, Indexes and query plans;
- milestone **Microservices**, 6/8: Saga in microservices ✓, Message Queue benefits ✓, Ordering
  problem review ✓, gRPC connection load balancing ✓, RabbitMQ prefetch ✓, DLQ setup, gRPC
  deadline propagation ✓, Kafka review;
- no personal data beyond the persona's name, no long notes. Every tile that shows a step,
  milestone or count draws it from this file, so start-state counts agree across tiles. `GATEWAY_TASK` is
  removed with the last mock that uses it.

**R15** — Tile content, start state → end state (the end state is what a hover reaches):

| Tile | Start state | Reveal (end state) |
|---|---|---|
| PLAN | Both milestone cards with progress bars; Database at 1/5, Microservices at 6/8; a few items each visible | One open Database step ticks (1/5 → 2/5) and its bar fills to match; further items reveal in the cards |
| TODAY | A Day plan: a few AI-suggested entries drawn from the open steps above, grouped under short labels with rough time estimates, a day meter partly filled | Ticks land on entries one after another; the day meter fills to match |
| REVIEW | End-of-day chat bubbles: Chris's short note on the day and one reply | The next reply slides in |
| MILESTONES | Plan overview (title, "N of 14 milestones") over a 14-segment progress strip, each segment filled to its milestone's progress; Database and Microservices among the 14 at 1/5 and 6/8 | Segments fill one by one (staggered) |
| CALENDAR | A month card with dated items from the plan | One day lights up |

A changing number (1/5 → 2/5, "N of 14") is two pre-rendered labels cross-faded, not a text
change, so the reveal stays transform/opacity only.

### Reveal contract

**R16** — Every reveal animates `transform` and `opacity` only (bars and segments fill by
`scaleX` from the start edge). No layout-triggering property is animated, and no layout is read
per frame.

**R17** — Triggers:
1. **Hover-capable pointer** (`(hover: hover)`): pointer over the tile plays the reveal; pointer
   out returns it to the start state. Lives in the landing CSS module under
   `@media (hover: hover)`.
2. **Keyboard:** `:focus-visible` on the tile link plays the reveal exactly as hover does, on
   every device.
3. **No hover** (`(hover: none)`, touch): the reveal plays **once**, when the tile first scrolls
   into view, and the tile then stays in its end state. A tile scrolled or flung past without an
   intersecting frame lands in its end state too (the `RevealOnEnter` fling rule).
4. **`prefers-reduced-motion: reduce`:** no reveal plays; every tile renders its end state
   statically from first paint after hydration, on every device.

**R18** — The tile's state is exposed as a data attribute on the shell (e.g.
`data-reveal="start|playing|end"` or equivalent), so tests can assert the contract without
computed styles. Server render and no-JS render the start state, fully legible.

**R19** — No content changes as a function of scroll position. No pinning, no scroll hijacking,
no horizontal scroll; page scroll length equals visual length (restates FS-0003 R9).

**R20** — Tile entrance: the grid fades and rises in on enter via the existing `riseIn` (through
`Section`/`RevealOnEnter`). The `fadeIn` keyframe stays unused by the landing. Tile reveal
keyframes, if any, are landing-local in the CSS module.

**R21** — Depth: the 0.2× ember glow behind the hero and the close stays as FS-0003 R7/R10 define
it. Tiles and their mocks are not parallaxed. `SplitSection` and the `ART_SPEED` layer are removed
with the sections that used them; `ParallaxLayer` stays for the glow.

### Close

**R22** — `ReturnSection` stays the last section, reprising the hearth glow, with its copy
unchanged and scaled up to sit with the new hero (headline ≈ `text-4xl`–`text-6xl`, `foreground`,
not a second coral moment). One solid "Start your plan" (`--action`) plus the quiet "Sign in"
text link (FS-0003 R5).

### Visual, access, behaviour

**R23** — Focus order runs eyebrow-free: badge → hero CTAs → the five tiles in DOM order → close
CTAs, with the chrome bar's controls reachable once it is visible. Each tile link's accessible
name is its label plus "Try it" (the arrow is `aria-hidden`). Mocks inside a tile are announced as
nothing.

**R24** — Token-only colour (FS-0003 R13): no raw hex, no `rgb(`/`hsl(` literals, no `*-gray-*`,
no `bg-white`/`text-white`/`bg-black` in the landing directory, the CSS module included (it may
use `hsl(var(--token))`). Dark mode at the token layer only.

**R25** — Both themes: light renders the x.ai tokens (white page, ivory tiles, hairlines, flat
tiles); dark renders the warm dark tokens. Mocks are crisp in both, with no blurry raster art.

**R26** — First paint is not gated on auth (FS-0003 R19). The hero and the tiles' start-state
content are in server-rendered markup.

**R27** — Stack conventions from `client/CLAUDE.md`: Tailwind v3, `cn()`, `next/link`, no `any`,
`"use client"` pushed as far down as practical.

### Cleanup

**R28** — Removed: `sections/PlanSection`, `DailySection`, `ReviewSection`, `NudgeSection` and
their tests; `SplitSection`; the mocks only they used (`ChecklistMock`, `PlanCardMock`,
`DailyReviewMock`, `SuggestionMock`, `CheckRow` unless a new mock reuses it); `ART_SPEED`.
Tests for the hero, chrome bar and close are updated to the new contract.

### Verification

**R29** — The final slice runs the client from this worktree on :3011 and captures screenshots
of `/` logged out in light and dark at desktop (≥1280px) and mobile (375px), including one hovered
or focused tile on desktop. Findings go in the PR, not in the repo.

## User Stories

1. As a first-time visitor, I want the first screen to say what Fireplace is in a few huge words,
   so that I get it before I scroll.
2. As a first-time visitor, I want the hero to feel clean and confident, so that the product
   reads as serious rather than as a side project.
3. As a first-time visitor, I want the hero to still feel warm, so that "sit down by the fire"
   isn't contradicted by a cold page.
4. As a first-time visitor, I want a clear "Start your plan" button in the hero, so that I can
   act the moment I'm convinced.
5. As a returning visitor with an account, I want "Sign in" visible in the hero, so that I don't
   have to hunt for it.
6. As a visitor, I want the "New" badge to take me to the Day plan, so that the newest feature is
   one click away.
7. As a visitor, I want to see the product's main features side by side, so that I can scan them
   in one glance instead of scrolling through a story.
8. As a visitor, I want each tile to look like the real screen, so that I know what I'd actually
   be using.
9. As a visitor, I want to hover a tile and see it do something, a box ticking or a bar filling,
   so that I feel how the feature works.
10. As a visitor, I want the plan tile to show milestones with real-looking steps, so that I see
    plans are more than a to-do list.
11. As a visitor, I want the Today tile to show an AI-picked list for the day, so that I see the
    AI does the boring part of deciding.
12. As a visitor, I want the Today tile to stand out, so that I notice the headline feature.
13. As a visitor, I want the review tile to show a short end-of-day exchange, so that I know the
    product expects reflection.
14. As a visitor, I want the milestones tile to show progress across a whole plan, so that I see
    the long arc, not just today.
15. As a visitor, I want the calendar tile to show dated work, so that I know plans can carry
    dates.
16. As a visitor, I want every tile to tell the same story about one person and one plan, so that
    the features read as one product.
17. As a visitor, I want to click any tile to start, so that interest turns into sign-up with no
    dead end.
18. As a keyboard visitor, I want tabbing onto a tile to play the same reveal as hovering, so that
    I don't miss what mouse users see.
19. As a keyboard visitor, I want the tab order to follow the grid as I see it, so that focus
    never jumps around.
20. As a phone visitor, I want the tiles stacked in one column, so that nothing is cramped or
    scrolls sideways.
21. As a phone visitor, I want each tile to play its reveal once as I scroll to it, so that I get
    the "show more" without a hover.
22. As a tablet visitor, I want two columns, so that the space is used without squeezing the
    mocks.
23. As a visitor with reduced motion on, I want every tile already showing its end state, so that
    I get the content without the animation.
24. As a screen-reader user, I want each tile announced as a link with its feature name and "Try
    it", so that I'm not read a list of fake checkboxes.
25. As a visitor, I want the top bar to appear only after the hero, so that the first impression
    is uncluttered.
26. As a visitor partway down, I want "Sign in" and "Start your plan" in the top bar, so that I
    can act without scrolling back.
27. As a visitor, I want to switch theme from the landing, so that I see it in the mode I use.
28. As a dark-mode visitor, I want the landing as warm as the app's dark theme, so that the theme
    isn't an afterthought.
29. As a light-mode visitor, I want the landing to match the app's new light look, so that signing
    up doesn't land me somewhere that looks different.
30. As a visitor at the end of the page, I want one clear closing call to action, so that the page
    ends on a decision, not a list.
31. As a visitor on a slow connection, I want the hero and tile content readable before any
    script runs, so that the page is useful immediately.
32. As a search crawler, I want the headline, subhead and tile labels in the markup, so that the
    landing is indexable.
33. As a maintainer, I want the landing still sealed in its own directory, so that a product
    refactor can't break it and its motion can't leak into the app.
34. As a maintainer, I want all invented content in one file, so that changing the persona or a
    step updates every tile at once.
35. As a maintainer, I want one tile shell that owns the hover, focus, touch and reduced-motion
    rules, so that a new tile inherits them instead of reimplementing them.
36. As a designer, I want the landing built only from tokens, so that a palette change updates it
    for free.

## Acceptance Criteria

- [ ] Logged out, `/` renders chrome bar, hero, a grid of exactly five tiles (PLAN, TODAY, REVIEW,
      MILESTONES, CALENDAR, in that DOM order) and the close; nothing else.
- [ ] The four old story sections, their tests, `SplitSection`, `ART_SPEED` and the mocks only
      they used are gone.
- [ ] The hero shows the eyebrow logo + wordmark, the badge "New · Day plan with AI moves ↗", the
      `h1` "Start your plan now. Sit down by the fire." on two lines, the existing description
      verbatim, and the two pill CTAs.
- [ ] The `h1` renders at ≥74px (`text-7xl`) at `lg` and ≥50px below `md`, in `foreground`, with a
      coral underline on "fire"; there is exactly one `h1` on the page.
- [ ] The badge is an in-page link to the Today tile and is keyboard operable.
- [ ] Hero "Start your plan" uses `bg-action`; hero "Sign in" uses `bg-muted`; both are
      `rounded-control`.
- [ ] No chrome bar at scroll 0; after the hero it shows logo, theme toggle, "Sign in" and "Start
      your plan" (in that order on the right), and no profile menu.
- [ ] Every tile is a single link to `/auth?tab=signup` with an accessible name of its label plus
      "Try it", a label bottom-left and "Try it →" bottom-right.
- [ ] Tiles use `rounded-tile`, `bg-card`, `border-border`; TODAY uses `bg-secondary
      text-secondary-foreground` in both themes.
- [ ] At `lg`: PLAN, TODAY, REVIEW share row 1 equally; MILESTONES spans two columns and CALENDAR
      one in row 2. At `md`: two columns, CALENDAR spans both. Below `md`: one column.
- [ ] Visual order equals DOM order at every breakpoint; tab order runs badge, hero CTAs, the five
      tiles, the close CTAs.
- [ ] No horizontal overflow from 320px to 2560px wide, in both themes.
- [ ] PLAN shows Database 1/5 and Microservices 6/8 with the R14 steps; MILESTONES shows a
      14-segment strip with "N of 14 milestones"; start-state counts agree across tiles and come from
      `mocks/content.ts`.
- [ ] Hover (on a hover-capable pointer) and `:focus-visible` each move a tile from start to end
      state; leaving returns it to start.
- [ ] With `(hover: none)`, each tile plays once on first entering the viewport and stays in its
      end state; a tile flung past without an intersecting frame is in its end state.
- [ ] Under `prefers-reduced-motion: reduce`, every tile shows its end state statically and no
      reveal animation runs.
- [ ] The tile's state is readable from a data attribute; server render and no-JS show the start
      state with all content legible.
- [ ] A tile's box does not change size between start and end state.
- [ ] The landing's CSS module animates only `transform` and `opacity`; `grep` finds no animated
      `width`, `height`, `top`, `left`, `margin` or `padding` in it.
- [ ] Tile reveal keyframes and hover rules live only in the landing CSS module; `globals.css`,
      `tailwind.config.js` and the other x.ai-pass files are unmodified (`git diff --stat` shows
      no change to them).
- [ ] `grep` over `client/src/components/landing/` finds no raw hex, no `rgb(`/`hsl(` literal
      other than `hsl(var(--…))`, no `*-gray-*`, no `bg-white`/`text-white`/`bg-black`.
- [ ] `expectMockContract` passes for every tile; no mock imports a product component from
      `@/components`, consumes context, or fetches.
- [ ] The close keeps its copy, is larger, shows one `--action` "Start your plan" plus a quiet
      "Sign in" link, and reprises the ember glow.
- [ ] The headline, subhead, tile labels and tile copy are in server-rendered markup.
- [ ] Screenshots on :3011 in light and dark, at desktop and 375px mobile, including a hovered or
      focused tile on desktop, are attached to the PR.
- [ ] `npm run lint`, `npm test` and `npm run build` pass in `client/`.

## Edge States

| Condition | Expected |
|---|---|
| **Auth still resolving** | Hero and grid paint immediately; nothing waits on `isLoading`. |
| **Authenticated visitor at `/`** | Signed-in home replaces the landing once auth resolves; a brief hero flash is accepted (FS-0003 R19). |
| **JS disabled / pre-hydration** | All copy, tiles and CTAs present and usable in the start state; hover still works on desktop because it is CSS. Touch visitors see the start state. |
| **Touch device** | No hover state ever sticks; each tile plays once on entering view, then rests in its end state. A tap on a tile navigates to sign-up; it does not first "hover". |
| **Hybrid device (touch screen + mouse)** | Follows `(hover: hover)` as the browser reports it; `:focus-visible` works either way. |
| **Reduced motion, any device** | End state, static, from hydration on. |
| **Reduced motion toggled mid-visit** | Tiles snap to end state; nothing is left mid-animation or hidden. |
| **Fast hover across several tiles** | Each tile animates independently and returns to start on leave; no tile is left stuck between states. |
| **Keyboard focus then mouse hover elsewhere** | The focused tile stays revealed while focused; the hovered one reveals too. |
| **Rapid scroll / fling to bottom** | Grid and close render at rest; touch tiles skipped past are in their end state. |
| **Back-navigation from `/auth`** | Page restores coherently; already-entered content at rest; touch tiles that already played stay in end state, or replay once. Never hidden. |
| **Theme toggled while a tile is revealed** | Colours swap in place; reveal state is kept; no replay. |
| **Narrow phone (320px)** | Headline wraps to more lines; no overflow; tiles stack, mocks scale down without clipping their labels. |
| **Very wide viewport** | Grid stays at the container's max width, centred; the glow scales without seams. |
| **200% zoom** | Layout falls back toward fewer columns; nothing clips; no horizontal scroll. |
| **Badge clicked under reduced motion** | Jumps to the Today tile without smooth scroll. |
| **Print / reader mode** | Copy degrades to readable text; mocks may print in their start state. |

## Out of Scope

- Any backend or HTTP work. No endpoints are added or changed, so there is no `§API surface` and
  ADR-0002 serialize-on-touch does not apply.
- Real data, screenshots, or any fetch on the landing.
- Public feature pages per tile.
- Interactivity beyond navigation and the reveal: no try-before-signup, no local checklist.
- The `/auth` page and the signed-in home.
- Any edit to the x.ai light-theme pass files, including `globals.css`, `tailwind.config.js`,
  `ThemeToggle.tsx` and `utils.ts`. New tokens, if one is ever needed, are that pass's call.
- Reconciling `client/docs/design-guideline.md` with the x.ai pass (white page, Lora,
  non-coral `h1`).
- Rewording the anchor phrase, the description, or the close's copy.
- Editing FS-0003. It is shipped and write-once; this FS records the supersession.

### Alternatives rejected

| Alternative | Why rejected |
|---|---|
| Real product screenshots in tiles | Blur at scale, stale on the next UI change, can't follow the theme, can't animate a real tick (D2) |
| Per-tile public feature pages as link targets | None exist (D7) |
| Keeping the four story sections under the grid | The grid replaces them (D5) |
| Tailwind `hover:` utilities for the reveal | Not gated by `(hover: hover)` in Tailwind v3, so touch would get sticky hover states (R17) |
| Parallax on tile art | Hover targets that drift from the pointer feel broken (derived, R21) |
| `grid-flow-dense` to fill the `md` gap | Visual order would diverge from tab order (R10) |
