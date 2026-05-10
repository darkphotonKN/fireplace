# Fireplace — Product Specification

**Version:** 1.0  
**Type:** Productivity Platform (Task Management + AI Insights + Learning Resource Discovery)  
**One-liner:** A developer and learning platform that houses your checklists, daily reviews, learning tracks, and AI-powered suggestions — the hearth of your productivity.

---

## Domain Terms

| Term              | Definition                                                                                                       |
| ----------------- | ---------------------------------------------------------------------------------------------------------------- |
| Plan              | A project or learning goal container — type: `project` or `learning`                                          |
| Checklist Item    | A task within a plan, scoped as `daily` (recurring) or `longterm` (milestone)                                    |
| Daily Reset       | Automatic nightly reset of completed daily-scoped items to `done=false`                                          |
| Focus             | A plan's stated objective — fed to AI for context-aware suggestions                                              |
| Scope             | Whether a checklist item is `daily` (short-term, repeatable) or `longterm` (milestone-based)                     |
| Insights          | AI-generated suggestions — checklist item suggestions, daily suggestions, video recommendations                  |
| Resource          | A learning material (video, article, GitHub repo) linked to a plan                                               |
| Discovery         | The system that crawls YouTube to find relevant tutorial videos based on plan focus                              |
| Content Generator | Interface wrapping OpenAI GPT-4o for generating suggestions and search terms                                     |
| Plan Calendar     | Per-plan Gantt-style calendar showing checklist items with dates — week or month view                            |
| Gantt Bar         | A horizontal bar spanning a checklist item's `start_date` to `due_date` on the calendar                          |
| Single-Day Chip   | A compact one-day marker shown when a checklist item has only `start_date` or `due_date` set (or both equal)     |
| Task              | A checklist item with `type='task'` (default) — supports the done checkbox                                       |
| Note              | A checklist item with `type='note'` — text-only entry; no checkbox, `done` always false                          |
| Parent Item       | A top-level checklist item (`parent_id IS NULL`) that may have children                                          |
| Child Item        | A nested checklist item whose `parent_id` references its parent                                                  |
| Two-Tier Nesting  | Parents may have children, but children may not — depth is capped at exactly two tiers                           |

---

## Features (Implemented)

> These features are already built. Listed here for context — do not reimplement.

### Authentication & Users

- [x] User registration and login with email/password (JWT access + refresh tokens)
- [x] Password hashing with bcrypt
- [x] Seed data: test user and test plan for development

### Plans

- [x] CRUD for plans (create, read, update, delete)
- [x] Plan types: `project` or `learning`
- [x] Focus field — primary objective text fed to AI
- [x] Daily reset toggle per plan — controls whether daily items auto-reset
- [x] Toggle daily reset endpoint

### Checklist Items

- [x] CRUD for checklist items within a plan
- [x] Scoped as `daily` or `longterm`
- [x] Auto-incrementing sequence for ordering
- [x] Scheduled time support (optional future datetime)
- [x] Archive/unarchive items
- [x] Upcoming items query (items with scheduled_time within next week)
- [x] Bulk reset of daily items (background job, respects plan's daily_reset flag)

### AI Insights

- [x] Single checklist suggestion — AI suggests next actionable task based on focus + existing items
- [x] Daily suggestions — 3 AI-generated daily tasks breaking down longterm items
- [x] Video suggestions — AI generates search terms, YouTube crawler finds relevant tutorials
- [x] Search term generator with dedicated system prompt
- [x] Content generator interface (pluggable, currently OpenAI GPT-4o)

### Background Jobs

- [x] Daily reset job — cron (14:00 UTC daily), resets all daily-scoped done items where plan has daily_reset=true
- [x] Scheduled items job — checks every minute for items with approaching scheduled_time
- [x] Job manager — start/stop lifecycle for all background jobs

### Discovery (YouTube Crawler)

- [x] Basic web crawler with configurable base URL
- [x] YouTube search result parsing via regex (extracts video IDs from raw HTML)
- [x] Concurrent crawling (3 goroutines for 3 search terms)
- [x] Resource model with title, URL, source, type, description

---

## Features (Current — In Progress)

### Plan Calendar (Gantt)

- [ ] BE: Migration — add `start_date` (DATE, nullable) and `due_date` (DATE, nullable) columns to `checklist_items`
- [ ] BE: Migration — backfill `start_date` from existing `scheduled_time` (date portion), then drop `scheduled_time`
- [ ] BE: Migration — drop `calendar_entries` table (superseded; no slot/pin model)
- [ ] BE: GET `/api/plans/:id/calendar?view=month&date=2026-03` — returns checklist items whose `[start_date, due_date]` range intersects the requested window (also supports `view=week`)
- [ ] BE: PATCH `/api/plans/:id/checklists/:checklist_id/dates` — update `start_date` and/or `due_date`, validates `start_date <= due_date` when both set
- [ ] BE: Remove `PATCH /api/plans/:id/checklists/:checklist_id/schedule` (replaced by `/dates`)
- [ ] FE (`flow-client`): Per-plan calendar page with toggle between week view and month view
- [ ] FE: Gantt-style horizontal bars span across day cells from `start_date` to `due_date` (inclusive)
- [ ] FE: Single-day chip rendered when only one of `start_date` / `due_date` is set, or when they are equal
- [ ] FE: Drag bar middle to move — both dates shift by the same delta, preserves duration
- [ ] FE: Drag left edge to resize `start_date`; drag right edge to resize `due_date`
- [ ] FE: Drag chip to move single date by delta
- [ ] FE: All drag/resize interactions persist via PATCH `/dates`
- [ ] FE: Only checklist items with at least one date set appear on the calendar

> Not in scope: AI auto-scheduling, slot caps, backlog sidebar, cross-plan view, pinning

### Nested Items + Notes

- [ ] BE: Migration — add `type` (TEXT, NOT NULL, DEFAULT 'task' — CHECK IN ('task','note')) and `parent_id` (UUID, nullable, FK → `checklist_items(id)` ON DELETE CASCADE) to `checklist_items`
- [ ] BE: Migration — add CHECK constraint `(type = 'task') OR (done = false)` so notes can never be done
- [ ] BE: Migration — add trigger that rejects INSERT/UPDATE where `parent_id` references a row whose own `parent_id` is NOT NULL (enforces two-tier max)
- [ ] BE: Existing `POST /api/plans/:id/checklists` accepts optional `type` and `parent_id`
- [ ] BE: Existing `PATCH /api/plans/:id/checklists/:checklist_id` accepts optional `type` and `parent_id` — setting `parent_id` to a value indents, setting to null outdents
- [ ] BE: Service-layer validation — `parent_id` (when set) must reference a row in the same plan; setting `parent_id` on a row that already has children is rejected (would create a third tier)
- [ ] BE: `GET /api/plans/:id/checklists?type=task|note` filter
- [ ] BE: Service-layer rule — daily-scoped tasks may only be created via the AI suggestion accept flow; manual creation of `scope='daily'` items is rejected
- [ ] FE (`flow-client`): Layout — left side is the Checklist block with tabs `All | Notes | Checklist` (longterm items, both types); right side is the Daily block (replaces the previous Note block position)
- [ ] FE: Daily block lists AI-suggested daily tasks only; no manual "add" affordance; existing daily reset logic unchanged
- [ ] FE: Note block hidden for now (preserved for a future feature)
- [ ] FE: Item row icons replace inline controls — type toggle (task ↔ note), indent (right arrow), schedule (calendar), stash (archive)
- [ ] FE: Type toggle — switching to `note` hides the checkbox and forces `done=false`; switching to `task` shows the checkbox
- [ ] FE: `Tab` while editing/highlighted → indent under the item above (PATCH `parent_id` = previous-sibling's id)
- [ ] FE: `Shift+Tab` while editing/highlighted → outdent to top level (PATCH `parent_id` = null)
- [ ] FE: Indent icon on the row triggers the same flow as `Tab`
- [ ] FE: Children render visually indented under their parent
- [ ] FE: Indent rejected if the row above is itself a child (max two tiers); UI no-ops or shows a brief message

> Not in scope: drag-to-nest, more than two tiers, parent-delete UX (warning dialogs / undo — DB cascades unconditionally for now), restoring the dedicated Note block

### Frontend Auth (`flow-client`)

- [ ] FE: `/auth` page with tab toggle between "Sign In" and "Sign Up"
- [ ] FE: Sign In form — email, password, submit
- [ ] FE: Sign Up form — name, email, password, confirm password, submit
- [ ] FE: Inline field validation errors (empty fields, password mismatch, email format)
- [ ] FE: On success — store JWT tokens (accessToken, refreshToken) in localStorage, redirect to dashboard
- [ ] FE: Auth guard on all routes — no token → redirect to `/auth`
- [ ] FE: `/auth` redirects to dashboard if already authenticated
- [ ] FE: Attach `Authorization: Bearer <token>` to all API requests
- [ ] FE: On 401 response — attempt token refresh, if fails → clear tokens → redirect to `/auth`

> Uses existing BE endpoints: POST `/api/users/signup`, POST `/api/users/signin`. No backend changes.
> Not in scope: forgot password, OAuth, email verification, profile page

### User Profile

- [ ] BE: Migration — add `display_name` and `bio` columns to users table
- [ ] BE: GET `/api/users/profile` — return own profile using JWT identity
- [ ] BE: PATCH `/api/users/profile` — update own profile (display_name, bio, name)
- [ ] FE (`flow-client`): Header dropdown shows real user name (display_name, falls back to name)
- [ ] FE (`flow-client`): Header dropdown with "Profile" and "Sign Out" options
- [ ] FE (`flow-client`): Dedicated `/profile` page with inline editing (click to edit, save button)
- [ ] FE (`flow-client`): Profile fields — display_name (editable), name (editable), email (read-only), bio (editable)

> Not in scope: avatar, timezone, notifications, public profiles, password change

### Authorization Middleware + Landing Page

- [ ] BE: Gin auth middleware — extract `user_id` from JWT `sub` claim, inject into `gin.Context`
- [ ] BE: Apply middleware to all routes except POST `/api/users/signup` and POST `/api/users/signin`
- [ ] BE: Remove all hardcoded `"11111111-1111-1111-1111-111111111111"` — read user ID from context
- [ ] BE: 401 for missing/expired token, 403 for ownership violations
- [ ] BE: Plans — add `WHERE user_id = ?` from JWT on all queries (list, get, update, delete)
- [ ] BE: Checklist items — verify plan belongs to authenticated user before any operation
- [ ] BE: Insights — verify `plan_id` belongs to authenticated user before generating
- [ ] FE (`flow-client`): `/` root shows splash page for unauthenticated users
- [ ] FE (`flow-client`): Splash tagline "Start your plan now. Sit down by the fire." with coral theme
- [ ] FE (`flow-client`): Sign In / Sign Up buttons linking to `/auth`
- [ ] FE (`flow-client`): Authenticated users on `/` redirect to dashboard

> Not in scope: RBAC, admin roles, sharing, public plans

### Light Mode Toggle (`flow-client`)

- [ ] FE: Sun/moon icon in header, positioned between existing content and username dropdown
- [ ] FE: Click toggles between dark mode and light mode
- [ ] FE: Dark mode (existing) — orange accent + dark brown/warm blacks
- [ ] FE: Light mode (new) — same orange accent + warm cream/off-white/parchment backgrounds, no cold whites
- [ ] FE: Theme preference saved in localStorage (`theme` key), defaults to `dark`
- [ ] FE: Apply theme class to `<html>` element on load to prevent flash of wrong theme
- [ ] FE: All pages and components respect the active theme

> Frontend-only, no backend changes.
> Not in scope: system preference detection, backend storage, per-component overrides

### Toast Notifications + Sidebar Hints (`flow-client`)

- [ ] FE: Reusable toast component — warm fireplace theme, theme-aware (dark/light)
- [ ] FE: Default position bottom-right, configurable bottom-left for sidebar hints
- [ ] FE: Auto-dismiss after 5 seconds, manual close (X button)
- [ ] FE: Vertical stacking when multiple toasts fire simultaneously
- [ ] FE: Toast on successful login — "Welcome back, {name}" (bottom-right)
- [ ] FE: Toast on profile updated — "Profile updated" (bottom-right)
- [ ] FE: Toast on plan created — "Plan created" (bottom-right)
- [ ] FE: Sidebar hint (first-timer) — "Tip: Your plans live in the side panel ←" (bottom-left, once)
- [ ] FE: Sidebar reminder (inactive 24hrs) — "Tip: Switch plans from the side panel ←" (bottom-left)
- [ ] FE: localStorage tracking: `hasSeenSidebarHint`, `lastSidebarOpen` timestamp
- [ ] FE: Fix sidebar collapse button overlapping Projects/Learning toggle

> Frontend-only, no backend changes.
> Not triggered by: checklist CRUD, scheduling, archiving, or any in-plan micro-actions.
> Not in scope: sound effects, notification history, backend-driven notifications, push notifications

---

## Features (Future — Not Started)

- [ ] Analytics dashboard — hybrid real-time/batch processing for task completion rates, streaks, active plans
- [ ] Unified cross-plan calendar view
- [ ] Time-blocking / hour-level scheduling
- [ ] AI auto-scheduling (suggest dates for items based on plan focus and existing schedule)
- [ ] AI-generated daily focus narrative and reflection prompts
- [ ] Backlog sidebar — dateless items panel alongside the calendar
- [ ] Pinning / locking items to specific dates
- [ ] Google Calendar / external calendar sync
- [ ] Workspace / music integration
- [ ] Real-time notifications for date-due items

---

## Data Model

### Users

```
users
├── id (PK, UUID, auto-generated)
├── name (TEXT, NOT NULL)
├── email (TEXT, UNIQUE, NOT NULL)
├── password (TEXT, NOT NULL, bcrypt hash)
├── display_name (TEXT, nullable)
├── bio (TEXT, nullable)
├── created_at (TIMESTAMP)
└── updated_at (TIMESTAMP)
```

### Plans

```
plans
├── id (PK, UUID, auto-generated)
├── user_id (FK → users, ON DELETE CASCADE)
├── name (TEXT, NOT NULL)
├── focus (TEXT, NOT NULL)
├── description (TEXT)
├── plan_type (TEXT, NOT NULL — 'project' | 'learning')
├── daily_reset (BOOLEAN, NOT NULL, DEFAULT true)
├── created_at (TIMESTAMPTZ)
└── updated_at (TIMESTAMPTZ, trigger-updated)
```

### Checklist Items

```
checklist_items
├── id (PK, UUID, auto-generated)
├── plan_id (FK → plans, ON DELETE CASCADE)
├── parent_id (UUID, nullable, FK → checklist_items(id), ON DELETE CASCADE — two-tier max enforced by trigger)
├── description (TEXT, NOT NULL)
├── done (BOOLEAN, NOT NULL, DEFAULT false)
├── sequence (INTEGER, NOT NULL)
├── type (TEXT, NOT NULL, DEFAULT 'task' — CHECK IN ('task', 'note'))
├── scope (TEXT, NOT NULL, DEFAULT 'longterm' — CHECK IN ('daily', 'longterm'))
├── start_date (DATE, nullable — Plan Calendar Gantt range start)
├── due_date (DATE, nullable — Plan Calendar Gantt range end; CHECK start_date <= due_date when both set)
├── archived (BOOLEAN, DEFAULT false)
├── created_at (TIMESTAMPTZ)
└── updated_at (TIMESTAMPTZ, trigger-updated)

Constraints:
  CHECK (type = 'task' OR done = false)        — notes can never be done
  CHECK (start_date IS NULL OR due_date IS NULL OR start_date <= due_date)
  TRIGGER on INSERT/UPDATE: if parent_id IS NOT NULL, reject when the
    referenced row's own parent_id IS NOT NULL (two-tier depth max)
Indexes:
  (plan_id, parent_id) — fast "children of X" / "top-level items in plan" queries
```

### Resources

```
resources
├── id (PK, UUID, auto-generated)
├── plan_id (FK → plans, ON DELETE CASCADE)
├── resource_type (TEXT, NOT NULL — e.g. 'Github', 'Youtube', 'Udemy')
├── url (TEXT, NOT NULL)
├── title (TEXT, NOT NULL)
├── description (TEXT, nullable)
├── sequence (INTEGER, NOT NULL)
├── created_at (TIMESTAMPTZ)
└── updated_at (TIMESTAMPTZ, trigger-updated)
```

### Plan Shares

```
plan_shares
├── user_id (PK, FK → users, ON DELETE CASCADE)
├── plan_id (PK, FK → plans, ON DELETE CASCADE)
├── role (TEXT, NOT NULL, DEFAULT 'edit' — CHECK IN ('edit', 'view'))
├── created_at (TIMESTAMPTZ)
└── updated_at (TIMESTAMPTZ, trigger-updated)

Indexes:
  PRIMARY KEY (user_id, plan_id) — composite key, one share per user per plan
  (user_id) — fast lookup: "which plans are shared with me?"
  (plan_id) — fast lookup: "who has access to this plan?"
```

---

## API Surface

### User API

| Method | Endpoint             | Auth | Description                                  |
| ------ | -------------------- | ---- | -------------------------------------------- |
| POST   | `/api/users/signup`  | None | Register new user                            |
| POST   | `/api/users/signin`  | None | Login → JWT access + refresh tokens          |
| GET    | `/api/users/profile` | JWT  | Get own profile (from JWT identity)          |
| PATCH  | `/api/users/profile` | JWT  | Update own profile (display_name, bio, name) |
| GET    | `/api/users/:id`     | JWT  | Get user by ID                               |
| GET    | `/api/users`         | JWT  | List all users                               |

### Plan API

| Method | Endpoint                            | Auth | Description                            |
| ------ | ----------------------------------- | ---- | -------------------------------------- |
| GET    | `/api/plans`                        | JWT  | List all plans for user                |
| GET    | `/api/plans/:id`                    | JWT  | Get plan by ID                         |
| POST   | `/api/plans`                        | JWT  | Create plan                            |
| PATCH  | `/api/plans/:id`                    | JWT  | Update plan (name, focus, description) |
| PATCH  | `/api/plans/:id/toggle-daily-reset` | JWT  | Toggle daily reset setting             |
| DELETE | `/api/plans/:id`                    | JWT  | Delete plan                            |

### Checklist API

| Method | Endpoint                                           | Auth | Description                                                       |
| ------ | -------------------------------------------------- | ---- | ----------------------------------------------------------------- |
| GET    | `/api/plans/:id/checklists`                        | JWT  | List items (?scope=daily\|longterm, ?type=task\|note)             |
| GET    | `/api/plans/:id/checklists/archived`               | JWT  | List archived items                                               |
| GET    | `/api/plans/:id/checklists/upcoming`               | JWT  | List upcoming scheduled items                                     |
| GET    | `/api/plans/:id/checklists/:checklist_id`          | JWT  | Get single item                                                   |
| POST   | `/api/plans/:id/checklists`                        | JWT  | Create item — body may include `type`, `parent_id`                |
| PATCH  | `/api/plans/:id/checklists/:checklist_id`          | JWT  | Update item — body may set `type`, `parent_id` (indent / outdent) |
| DELETE | `/api/plans/:id/checklists/:checklist_id`          | JWT  | Delete item (cascades to children)                                |
| PATCH  | `/api/plans/:id/checklists/:checklist_id/dates`    | JWT  | Update start_date / due_date (drag)                               |
| PATCH  | `/api/plans/:id/checklists/:checklist_id/archive`  | JWT  | Archive item                                                      |

### Insights API

| Method | Endpoint                                             | Auth | Description                      |
| ------ | ---------------------------------------------------- | ---- | -------------------------------- |
| GET    | `/api/insights/checklist-suggestion?plan_id=X`       | JWT  | AI suggests next task            |
| GET    | `/api/insights/checklist-suggestion-daily?plan_id=X` | JWT  | AI suggests 3 daily tasks        |
| GET    | `/api/insights/suggest-videos?plan_id=X`             | JWT  | AI finds relevant YouTube videos |

### Calendar API (Plan Calendar — Gantt)

| Method | Endpoint                                                | Auth | Description                                                                  |
| ------ | ------------------------------------------------------- | ---- | ---------------------------------------------------------------------------- |
| GET    | `/api/plans/:id/calendar?view=month&date=2026-03`       | JWT  | Checklist items whose `[start_date, due_date]` intersects window (week/month) |

---

## Plan Calendar — Detailed Design

### Date Model

`start_date` and `due_date` are independent, both nullable `DATE` columns on `checklist_items`. Together they define a checklist item's position on the Plan Calendar:

| State                                | Calendar Render                                  |
| ------------------------------------ | ------------------------------------------------ |
| Both null                            | Item does not appear on calendar                 |
| Only `start_date` set                | Single-day chip on `start_date`                  |
| Only `due_date` set                  | Single-day chip on `due_date`                    |
| Both set, `start_date == due_date`   | Single-day chip on that date                     |
| Both set, `start_date < due_date`    | Gantt bar spanning both dates inclusive          |
| Both set, `start_date > due_date`    | Invalid — rejected at API (400)                  |

### Render

- **Month view**: 7-column grid for the requested month; bars span horizontally across day cells
- **Week view**: 7-column grid for the week containing `date` query param
- Items appear only if at least one of `start_date` / `due_date` is set
- Multiple items on the same day stack vertically within the cell

### GET `/api/plans/:id/calendar?view=<week|month>&date=<YYYY-MM-DD|YYYY-MM>`

- Returns checklist items where `[start_date, due_date]` (treating null as the other date) intersects the window
- Window is computed from `view` + `date`:
  - `view=month`, `date=2026-03` → window is `2026-03-01..2026-03-31`
  - `view=week`, `date=2026-03-09` → window is the Sun..Sat week containing that date
- Excludes archived items

### PATCH `/api/plans/:id/checklists/:checklist_id/dates`

Request body — both fields optional, either may be `null` to clear:

```json
{ "startDate": "2026-03-10", "dueDate": "2026-03-14" }
```

- Persists `start_date` / `due_date`; absent fields leave existing values untouched
- Validates `start_date <= due_date` when both are set (and when one is set in the body and the other already exists in the DB)
- Returns the updated checklist item

### Drag-and-Drop Semantics (Frontend)

| Gesture                        | Effect                                                              |
| ------------------------------ | ------------------------------------------------------------------- |
| Drag bar middle by Δ days      | `start_date += Δ`, `due_date += Δ` (preserves duration)             |
| Drag bar left edge by Δ days   | `start_date += Δ`; `due_date` unchanged                             |
| Drag bar right edge by Δ days  | `due_date += Δ`; `start_date` unchanged                             |
| Drag single-day chip by Δ days | Whichever date is set shifts by Δ; if both equal, both shift by Δ   |

Each gesture commits via a single PATCH `/dates` call.

---

## Nested Items + Notes — Detailed Design

### Type

`type` is a `TEXT` column with values `'task'` or `'note'`. Default `'task'`.

| Type    | Checkbox rendered? | `done` allowed? | Use case                                |
| ------- | ------------------ | --------------- | --------------------------------------- |
| `task`  | yes                | true / false    | Actionable item (default for new rows)  |
| `note`  | no                 | always false    | Inline plain-text annotation in a list  |

The checkbox / `done` invariant is enforced by the DB CHECK
`(type = 'task' OR done = false)` and by the service layer on update.
Toggling a task with `done=true` to a note is rejected (or silently sets `done=false` first — implementation choice; service layer flips `done` to false on type=note transition for ergonomics).

### Two-Tier Nesting

`parent_id` is a self-referencing nullable FK on `checklist_items`. A row with `parent_id IS NULL` is a parent (top-level); a row with `parent_id` set is a child. Children may not have children.

Enforcement:

- **DB trigger** on INSERT/UPDATE: when `NEW.parent_id IS NOT NULL`, look up the referenced row; if its `parent_id IS NOT NULL`, raise an exception. Same trigger fires on UPDATE to a parent row's `parent_id` — a row with existing children cannot itself become a child (would push children to tier 3).
- **Service layer**: rejects requests violating the rule with HTTP 400 before hitting the DB, so the trigger is a backstop rather than the primary error path.

### Indent / Outdent

- **Indent**: `PATCH /api/plans/:id/checklists/:checklist_id` with body `{ "parent_id": "<sibling-uuid>" }`. The frontend computes the parent as the previous-sibling top-level item in render order.
- **Outdent**: same endpoint with body `{ "parent_id": null }`.
- Setting `parent_id` to a row in a different plan is rejected by the service layer (cross-plan nesting forbidden).

### Daily Items Source

After this feature ships, `scope='daily'` items can only be created via the AI suggestion accept flow (existing `GET /api/insights/checklist-suggestion-daily` → POST). Manual `POST /checklists` with `scope='daily'` is rejected by the service layer. Existing daily items are unaffected.

The Daily block on the FE has no manual "add" affordance; only a "Suggest" button that opens the existing AI suggestion list.

### Frontend Layout

| Region     | Content                                                                                  |
| ---------- | ---------------------------------------------------------------------------------------- |
| Left side  | Checklist block — longterm items, both types. Tab filter `All \| Notes \| Checklist`    |
| Right side | Daily block — AI-generated daily tasks; no manual create; existing daily reset preserved |
| (hidden)   | Note block — preserved in the layout system but not surfaced in this iteration          |

Item-row icons replace the previous inline controls:

| Icon          | Effect                                                                              |
| ------------- | ----------------------------------------------------------------------------------- |
| Type toggle   | Flip between `task` (checkbox) and `note` (no checkbox)                             |
| Indent arrow  | Same as `Tab` — nest under previous-sibling item                                    |
| Schedule      | Open date picker for `start_date` / `due_date` (PATCH `/dates`)                     |
| Stash         | Soft-delete via existing archive endpoint (PATCH `/archive`)                        |

---

## Business Rules

### Daily Reset

- Runs at 14:00 UTC daily via cron job
- Only resets items where `scope='daily'` AND `done=true` AND plan has `daily_reset=true`
- Uses bulk CTE update (not row-by-row)

### Checklist Scope Validation

- Scope must be `daily` or `longterm` — enforced by DB CHECK constraint and service-layer validation

### Plan Calendar Date Validation

- `start_date` and `due_date` are independent and both nullable
- When both are set, `start_date <= due_date` is enforced at API (400 on violation) and at DB level via CHECK constraint
- Setting one date does not require the other; clearing both removes the item from the calendar
- Window intersection (GET): an item appears if its `[start_date ?? due_date, due_date ?? start_date]` overlaps the requested view window

### Nested Items + Notes

- `type` defaults to `'task'`; `'note'` items render without a checkbox and have `done` permanently `false` (DB CHECK)
- Switching an item from `task` (with `done=true`) to `note` clears `done` to `false` server-side
- Two-tier nesting max — a child item cannot itself become a parent (DB trigger + service-layer guard)
- `parent_id` cannot reference a row in a different plan (service-layer guard)
- Deleting a parent cascades to its children via FK `ON DELETE CASCADE`; archiving a parent does NOT archive children
- `scope='daily'` items cannot be created manually after this feature ships — only via the AI suggestion accept flow
- Daily reset job (existing) is unchanged; it continues to operate on `scope='daily' AND type='task'` (notes are excluded by the DB CHECK above since they cannot be `done=true`)

### Frontend Auth Token Lifecycle

- Access token stored in `localStorage` as `accessToken`, refresh token as `refreshToken`
- All API requests include `Authorization: Bearer <accessToken>` header
- On 401 response: attempt refresh via POST `/api/users/refresh` with refreshToken, store new accessToken
- If refresh fails (401/expired): clear both tokens, redirect to `/auth`
- `/auth` page checks for existing valid token on mount — if present, redirect to dashboard

### Theme Toggle

- Theme stored in `localStorage` as `theme` with values `dark` or `light`, defaults to `dark`
- On app load: read `localStorage.theme`, apply class to `<html>` before first paint (inline script or layout effect)
- Dark mode: warm dark palette (`#1f1f1f` background, `#d1cfc0` text, `rgb(247,111,83)` accent)
- Light mode: warm light palette (cream/parchment backgrounds, dark warm text, same orange accent)
- Both modes maintain fireplace warmth — no cold grays or pure whites

### Authorization & Ownership

- Auth middleware runs on all routes except `/api/users/signup` and `/api/users/signin`
- Middleware extracts `user_id` from JWT `sub` claim and sets `gin.Context` key `userId`
- Missing or expired token → 401 Unauthorized
- Plan queries filter by `user_id` from JWT — user can only see/modify their own plans
- Checklist/calendar/insights operations verify the parent plan belongs to the authenticated user → 403 if not
- No cross-user data access is possible through any endpoint

### Toast Notification Triggers

- System-level events only — login, profile update, plan creation
- NOT triggered by checklist item CRUD, scheduling, archiving, or in-plan micro-actions
- Sidebar hint shown once per user (`hasSeenSidebarHint` in localStorage) on first plan page visit
- Sidebar reminder shown if user hasn't opened sidebar in 24hrs (`lastSidebarOpen` timestamp) and is on a plan page
- `lastSidebarOpen` updated each time user opens the sidebar
- Toasts auto-dismiss after 5 seconds unless manually closed
- Multiple simultaneous toasts stack vertically

---

## Edge Cases

### Plan Calendar

| Scenario                                          | Handling                                                                            |
| ------------------------------------------------- | ----------------------------------------------------------------------------------- |
| No checklist items have dates                     | Calendar renders empty grid                                                         |
| Item has only `start_date` (or only `due_date`)   | Single-day chip on the set date                                                     |
| `start_date == due_date`                          | Single-day chip                                                                     |
| PATCH sets `start_date > due_date`                | 400 — neither field persisted                                                       |
| Bar starts before window, ends inside             | Bar rendered clipped to left edge of window                                         |
| Bar starts inside window, ends after              | Bar rendered clipped to right edge of window                                        |
| Drag bar middle past month boundary (FE)          | Both dates still shift by Δ; calendar fetches next window if user navigates         |
| Drag edge past the other date (e.g. start > due)  | FE clamps to other date OR commits — backend rejects (400), FE reverts on error     |
| Item archived                                     | Excluded from calendar response regardless of dates                                 |
| Item deleted                                      | Disappears from calendar (no separate calendar row)                                 |

### Frontend Auth

| Scenario                         | Handling                                        |
| -------------------------------- | ----------------------------------------------- |
| Token missing from localStorage  | Redirect to `/auth`                             |
| Access token expired, refresh ok | Silently refresh, retry original request        |
| Both tokens expired              | Clear tokens, redirect to `/auth`               |
| Sign up with existing email      | Show API error message inline                   |
| Sign in with wrong password      | Show API error message inline                   |
| Password and confirm don't match | Client-side validation, block submit            |
| User manually navigates to /auth | If already authenticated, redirect to dashboard |

### Toast Notifications

| Scenario                                  | Handling                                                   |
| ----------------------------------------- | ---------------------------------------------------------- |
| Multiple toasts at same time              | Stack vertically, each auto-dismisses independently        |
| localStorage cleared (hasSeenSidebarHint) | Sidebar hint shows again on next plan page visit           |
| User opens sidebar then doesn't for 24hrs | Reminder toast shown on next plan page visit (bottom-left) |
| User never visits a plan page             | Sidebar hints never fire (only triggered on plan pages)    |
| Toast fired while another is dismissing   | New toast stacks, dismissing toast animates out            |

### Authorization & Ownership

| Scenario                                 | Handling                                    |
| ---------------------------------------- | ------------------------------------------- |
| No Authorization header                  | 401 Unauthorized                            |
| Expired JWT                              | 401 Unauthorized                            |
| Valid JWT but plan belongs to other user | 403 Forbidden                               |
| Checklist op on other user's plan        | 403 Forbidden (verify plan ownership first) |
| Insights request for other user's plan   | 403 Forbidden                               |
| Unauthenticated user visits `/`          | See splash/landing page                     |
| Authenticated user visits `/`            | Redirect to dashboard                       |

### Nested Items + Notes

| Scenario                                                   | Handling                                                                               |
| ---------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| Indent first item in list (no row above)                   | FE no-ops (nothing to nest under); backend never receives the request                  |
| Indent under a row that is itself a child                  | FE rejects pre-flight; if the request reaches the API, returns 400 (would be tier 3)   |
| Outdent a top-level item                                   | No-op (already at top); backend silently accepts `parent_id=null` on a null row        |
| Toggle a `task` with `done=true` to `note`                 | Server forces `done=false` first, then sets `type=note`; one PATCH commits both        |
| Toggle a `note` to `task`                                  | Type flips; `done` remains `false` until user checks the box                           |
| Delete a parent with children                              | Children cascade-delete (DB FK); FE refetches list                                     |
| Archive a parent                                           | Parent archived; children remain visible (archive does not cascade)                    |
| Manually POST `scope='daily'`                              | 400 — daily items must come from AI suggestions                                        |
| Filter `?type=note&scope=daily`                            | Returns empty (DB allows it but UX never creates such rows)                            |
| Filter tab `Notes` selected, no notes exist                | Empty state                                                                            |

### Concurrent Access

| Scenario                                            | Handling                                              |
| --------------------------------------------------- | ----------------------------------------------------- |
| Two PATCH /dates requests on the same item          | Last write wins (acceptable for single-user per plan) |
| GET calendar while another tab moves an item        | Stale view; refresh fetches latest state              |
| Two clients indent the same item simultaneously     | Last write wins; both end up with the same `parent_id` if targeting the same row |

---

## Non-Functional Requirements

| Category     | Requirement       | Target                                                                      |
| ------------ | ----------------- | --------------------------------------------------------------------------- |
| **Stack**    | Backend           | Go 1.23+ with Gin framework                                                 |
| **Stack**    | Database          | PostgreSQL (via sqlx)                                                       |
| **Stack**    | Frontend (future) | Next.js + shadcn + Tailwind + TypeScript                                    |
| **Stack**    | AI                | OpenAI GPT-4o via go-openai SDK                                             |
| **Stack**    | Migrations        | golang-migrate (sequential numbering)                                       |
| **Dev**      | Hot reload        | Air                                                                         |
| **Dev**      | DB                | Docker Compose (postgres:latest on port 5556)                               |
| **Auth**     | JWT access token  | 60 min expiry                                                               |
| **Auth**     | JWT refresh token | 7 day expiry                                                                |
| **Patterns** | Architecture      | Handler → Service → Repository (interface-based DI)                         |
| **Patterns** | Error handling    | errorutils.AnalyzeDBErr for all DB errors                                   |
| **Patterns** | Transactions      | dbutils.ExecTx wrapper                                                      |
| **Patterns** | DB queries        | Named queries with sqlx, context propagation                                |
| **Patterns** | Models            | Shared in `internal/models/entities.go`, request/response types per package |

---

## Package Structure

```
cmd/main.go                    — Entry point
config/
├── db.go                      — Database connection + pool config
├── migrations.go              — Migration runner + seed data
└── routes.go                  — Router setup + dependency wiring
internal/
├── ai/                        — OpenAI generators (checklist, search terms, daily focus)
├── auth/                      — JWT generation + refresh
├── calendar/                  — Plan Calendar (handler, service, repository, model — Gantt window queries)
├── checklistitems/            — Checklist CRUD + daily reset + scheduling
├── concepts/                  — Learning concept model
├── constants/                 — Shared constants, error types, enums
├── crawler/                   — (empty, placeholder)
├── discovery/                 — YouTube video finder + web crawler
├── insights/                  — AI suggestion orchestration (checklist + video)
├── interfaces/                — Shared interfaces (ContentGenerator)
├── jobs/                      — Background job manager + daily reset + scheduled items
├── models/                    — Shared entity models (User, Plan, ChecklistItem)
├── plans/                     — Plan CRUD + toggle daily reset
├── types/                     — Shared types
├── user/                      — User CRUD + auth
└── utils/
    ├── dbutils/               — Transaction helper
    └── errorutils/            — DB error analysis
migrations/                    — SQL migration files (currently 000001–000010)
```
