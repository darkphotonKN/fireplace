# Design Guideline — Flow Client

> **Single source of truth for all visual and design decisions** in the Flow Client.
> Any task that changes appearance — colours, typography, spacing, component styling, layout,
> motion, iconography, or theming — must read this file first and conform to it. When this file
> and any other doc (including `CLAUDE.md`) disagree on a visual decision, **this file wins.**

> Scope note: this is the design reference (the target look and the rules), not a changelog of
> the current code. It describes *what good looks like*. Where today's code deviates (see
> **Known drift** under Components), new work should move toward this doc, not extend the drift.

The living implementation of these tokens is [`../src/app/globals.css`](../src/app/globals.css)
(CSS variables) and [`../tailwind.config.js`](../tailwind.config.js) (scale + utilities). Those
files and this doc must stay in sync; the CSS variables are the runtime source, this doc is the
intent behind them.

---

## Identity & Mood

**Cozy · warm · focused.** Flow Client should feel like sitting down by the fire to plan your
day — the hearth of your productivity, not a cold dashboard. The aesthetic is *warm minimalism*:
soft cream and ember tones, a single confident coral accent, serif type for a calm, considered
feel. It should read as inviting and unhurried while still being crisp and legible enough for
daily task work.

Anchor phrase (product voice): *"Start your plan now. Sit down by the fire."*

---

## Design Principles

1. **Warmth over sterility.** No cold grays, no pure `#FFFFFF`, no pure black. Every neutral
   carries a warm (cream / ember / parchment) tint. If a surface looks clinical, it's wrong.
2. **Token-first, always.** Style from semantic tokens / CSS variables (`bg-background`,
   `text-foreground`, `border-border`, `bg-primary`…), never raw hex or `*-gray-*` utilities.
   Dark mode is handled at the token layer, never with per-component overrides.
3. **One coral moment per view.** The coral primary marks the single most important thing on a
   screen (primary action, active state, the `h1`). Overusing it kills its meaning.
4. **Clarity over density.** Generous spacing, one clear primary action, calm hierarchy. Legible
   before compact.
5. **Motion communicates, never decorates.** Animation exists to show *what changed* and *where*
   (a new item landing, a panel opening) — not for flourish.

---

## Color Palette & Tokens

Colours are defined as **HSL channel triplets** in CSS variables and consumed through Tailwind
tokens (e.g. `bg-primary`, `text-muted-foreground`). **The HSL values in `globals.css` are
authoritative**; hex values below are for reference/approximation.

### Brand

| Role | Token | HSL | Hex | Notes |
|---|---|---|---|---|
| Primary (coral/ember) | `--primary` / `primary` | `12 92% 65%` | `#F76F53` | The one accent. Buttons, active states, `h1`, focus ring. Identical in both themes. |
| Primary halo | — | `rgba(247,111,83,0.18)` | — | Soft glow used in the item-entrance animation. |

### Light theme (`:root`) — warm cream

| Role | Token | HSL | Hex (approx) |
|---|---|---|---|
| Background | `--background` | `55 43% 92%` | `#F2F0E3` |
| Foreground (text) | `--foreground` | `0 0% 18%` | `#2E2E2E` |
| Card / Popover | `--card` / `--popover` | `55 43% 92%` | `#F2F0E3` |
| Secondary (high-contrast) | `--secondary` | `0 0% 18%` | `#2E2E2E` (fg `#FFFFFF`) |
| Muted / Accent surface | `--muted` / `--accent` | `55 33% 87%` | `#E9E7D3` |
| Muted foreground | `--muted-foreground` | `0 0% 40%` | `#666666` |
| Border | `--border` | `12 30% 75%` | `#D8BCB2` (coral-tinted) |
| Input | `--input` | `55 33% 82%` | `#DED9C4` |
| Ring (focus) | `--ring` | `12 92% 65%` | `#F76F53` |
| Destructive | `--destructive` | `0 84% 60%` | `#EF4444` |
| Radius | `--radius` | — | `0.5rem` (8px) |

### Dark theme (`.dark`) — warm charcoal + ember

| Role | Token | HSL | Hex (approx) |
|---|---|---|---|
| Background | `--background` | `0 0% 12%` | `#1F1F1F` |
| Foreground (text) | `--foreground` | `43 10% 82%` | `#D1CFC0` |
| Card / Popover | `--card` / `--popover` | `0 0% 10%` | `#1A1A1A` |
| Muted | `--muted` | `0 0% 15%` | `#262626` |
| Muted foreground | `--muted-foreground` | `43 10% 60%` | `#9E9A8C` |
| Accent (ember) | `--accent` | `12 92% 15%` | `#491103` |
| Accent foreground | `--accent-foreground` | `43 10% 90%` | `#E9E6DC` |
| Border | `--border` | `12 15% 20%` | `#3B2E2B` (warm) |
| Input | `--input` | `0 0% 20%` | `#333333` |

> **Token gap to close:** `--primary-foreground` is consumed by Tailwind (`primary.foreground`)
> but is **not defined** in `globals.css`, so `text-primary-foreground` currently resolves to
> nothing. Define it as the on-coral text colour — the cream `#F2F0E3` (light) / near-white — so
> text on coral surfaces has a guaranteed contrast pair. Any new `--primary` usage should assume
> this exists.

### Usage rules
- Text on `background`/`card` → `foreground`; secondary text → `muted-foreground`.
- Coral (`primary`) is for emphasis and action, not large fills — a little goes a long way.
- Never introduce a new raw colour. If you need one, add a semantic token here + in `globals.css`
  for **both** themes.

---

## Typography

- **Family:** `Merriweather`, `Georgia`, serif — the serif is core to the cozy identity; do not
  swap to a sans for UI chrome. (Prefer wiring it via `next/font` over a raw `@import`.)
- **Base document size:** 14px (`body`). Weights in use: 300 / 400 / 700 / 900.
- **Utility scale is bumped +2px** from Tailwind defaults across the board (card titles, inputs,
  placeholders, body, badges all inherit it). Use the semantic `text-*` utilities, not px values:

| Utility | Size / line-height | Typical use |
|---|---|---|
| `text-xs` | 14 / 18 | captions, meta, badges |
| `text-sm` | 16 / 22 | secondary body, labels |
| `text-base` | 18 / 26 | body default |
| `text-lg` | 20 / 30 | lead text, `h3`-scale |
| `text-xl` | 22 / 30 | section titles |
| `text-2xl` | 26 / 34 | `h2`-scale |
| `text-3xl` | 32 / 38 | `h1`-scale / page titles |
| `text-4xl`+ | 38 / 42 → 130 | hero / splash |

### Heading rules (from base styles)
- **`h1` is always coral** (`#F76F53`), 32px / 700, in **both** light and dark. It is the brand's
  signature — one per view.
- `h2` 24 / 700, `h3` 20 / 700 — inherit `foreground` (cream text in dark).
- Body/`p` inherit `foreground`. Keep line length comfortable; serif needs breathing room.

---

## Spacing, Layout & Grid

- **Container:** centered, `2rem` horizontal padding, max width `1400px` at `2xl`.
- **Breakpoints (Tailwind):** `sm 640` · `md 768` · `lg 1024` · `xl 1280` · `2xl 1400`.
  Desktop/tablet-first; stay usable down to `sm`.
- **Spacing rhythm:** the workhorses are `px-4 py-2` for interactive controls and `p-6` for card
  padding; `gap-1`/`gap-2` for inline clusters, `gap-3`+ for grouped sections. Prefer the 4px
  step scale (`1,2,3,4,6`) — avoid arbitrary values.
- **Radius scale** (all derived from `--radius: 0.5rem`): `rounded-sm` 4px · `rounded-md` 6px ·
  `rounded-lg` 8px. Plus `rounded-full` for pills/avatars. Keep corners soft — sharp corners read
  as cold.

---

## Components & UI Chrome

Baseline is **shadcn/ui + Radix primitives**. Deviations must stay warm and token-driven.

- **Cards:** `rounded-lg bg-card text-card-foreground shadow-sm`, `p-6` content padding. The
  default elevation is subtle (`shadow-sm`); reserve `shadow-lg` for overlays/popovers.
- **Buttons:** `rounded-md`, medium weight, `transition-colors`, `focus-visible` ring.
  - *Primary* action → coral (`bg-primary` + on-coral foreground).
  - *Secondary* → high-contrast dark chip (`secondary`).
  - *Ghost/outline* → transparent with token border/hover.
  - Sizes: `h-8` (sm) · `h-9` (default) · `h-10` (lg) · `h-9 w-9` (icon).
- **Inputs / textareas:** `bg-input` (or `background`) with `border-border`, `rounded-md`, focus
  ring = `ring` (coral). Placeholders use `muted-foreground`.
- **Badges / pills:** `rounded-full`, `text-xs`, warm surface (`muted`/`accent`) — coral only for
  a genuinely active/selected pill.
- **Elevation:** shadows are soft and warm-neutral. Glass/`backdrop-blur` is an accepted accent
  for floating chrome (headers, overlays) — use sparingly, keep it subtle.
- **Borders:** the border token is intentionally **coral-tinted** (light) / warm-charcoal (dark).
  Don't replace with neutral gray borders.

> **Known drift (migrate toward tokens; don't add new):** the shadcn `button.tsx` variants and
> ~200 spots across `src/` still use hardcoded `*-gray-*` / `*-red-*` / `*-amber-*` / raw hex
> (e.g. button `default` is `bg-gray-700 text-gray-50`, focus ring `ring-gray-950`). These
> predate the token system and read cold. New/changed components must use tokens
> (`bg-secondary`, `text-muted-foreground`, `ring-ring`, …); when you touch a drifted component,
> retheme it as you go.

---

## Motion & Interaction

Defined in `tailwind.config.js` + `tailwindcss-animate`.

- **Hover/state transitions:** `transition-colors` (or `transition`), **~300ms default**. Nearly
  all interactive elements have a hover state — keep it a colour/opacity shift, not a jump.
- **Accordion / disclosure:** `0.2s ease-out` (`accordion-down`/`accordion-up`).
- **Item entrance (`animate-fadeIn`):** `600ms cubic-bezier(0.22, 1, 0.36, 1)` (ease-out-quint) —
  slides down 8px + fades in + a soft **coral halo pulse** (`0 0 0 4px rgba(247,111,83,0.18)`) so
  the user sees exactly where a new task landed. This is the signature motion; reuse the same
  curve + halo for "new thing appeared" moments rather than inventing new ones.
- **Feel:** snappy settle, never bouncy or slow. Durations 150–300ms for feedback, ~600ms max for
  entrances.
- Respect `prefers-reduced-motion`: reduce transforms/halo to a plain opacity fade.

---

## Theming (Dark / Light)

- **Mechanism:** `darkMode: ['class']` — the `.dark` class on the root toggles the theme; all
  colours come from CSS variables redefined under `.dark`. Preference persists in `localStorage`
  (`theme`, default `dark`) and is applied before first paint to avoid a flash.
- **Contract:** every colour a component uses must be a token that is defined in **both** `:root`
  and `.dark`. Never hardcode a dark-mode value in a component. Adding a token means adding it to
  both blocks.
- **Per-theme intent:** light = warm cream/parchment; dark = warm charcoal (`#1F1F1F`) with ember
  accents (`--accent` deep coral). The **coral `--primary` and the `h1` colour are identical in
  both themes** — the brand mark doesn't shift.
- Always verify a change in **both** themes before commit.

---

## Iconography & Imagery

- **Icon set:** [`lucide-react`](https://lucide.dev) exclusively — clean, rounded-stroke line
  icons that suit the warm, friendly tone. Don't mix in another icon library.
- **Sizing:** align to the type/spacing grid — `h-4 w-4` inline with text, `h-5 w-5` for standalone
  controls; icon buttons are `h-9 w-9`. Keep the default stroke width; scale with `size`, not
  transforms.
- **Colour:** icons inherit `currentColor` — they follow the text token. Coral only when the icon
  *is* the primary action/active state.
- **Imagery:** favour warmth and light (hearth/ember tones). Avoid cold stock or hard neutrals.

---

## Accessibility

- **Contrast:** target WCAG AA — 4.5:1 for body text, 3:1 for large text/UI. Watch coral-on-cream
  and `muted-foreground` on tinted surfaces; if it fails, darken the text token, don't lighten the
  brand. (Defining `--primary-foreground` closes the biggest gap — text on coral.)
- **Focus:** every interactive element shows a visible `focus-visible` ring; standardize on the
  coral `ring` token (replace the legacy gray focus rings as you touch them).
- **Keyboard:** full keyboard operability (the app advertises power-user shortcuts) — no
  mouse-only affordances; Radix primitives give correct roles/focus management, keep them.
- **Reduced motion:** honour `prefers-reduced-motion` (see Motion).

---

## Voice & Tone

- **Warm, encouraging, unhurried** — a calm companion, not a taskmaster. Lightly leans on the
  hearth/fireplace metaphor where it fits ("Sit down by the fire"), without overdoing it.
- **Concise and human.** Labels are short and plain; buttons are verbs ("Create plan", "Add
  task"). Sentence case, not Title Case, for UI copy.
- **Empty states** invite action rather than apologize ("No plans yet — start one by the fire").
- **Errors** are calm and constructive: what happened + what to do next, never blame.
- **Toasts** are brief and positive ("Welcome back, {name}", "Plan created").

---

## Out of Scope (not "design")

Changes that *look* like styling but are actually logic/architecture — do **not** alter these
under the banner of a visual pass:

- Data shapes, API request/response contracts, and the `src/services/` layer.
- Routing, page/component structure, and state management.
- Feature behaviour (what a control does), business rules, auth/token handling.
- **Token *values*** — changing a brand/semantic colour, the type scale, or `--radius` is a
  design-system decision, not a per-component style tweak; propose it as an edit to this doc +
  `globals.css`/`tailwind.config.js`, applied globally, not one-off in a component.
