# Fireplace — Product Specification

**Version:** 1.0  
**Type:** Productivity Platform (Task Management + AI Insights + Learning Resource Discovery)  
**One-liner:** A developer and learning platform that houses your checklists, daily reviews, learning tracks, and AI-powered suggestions — the hearth of your productivity.

---

## Domain Terms

| Term              | Definition                                                                                                       |
| ----------------- | ---------------------------------------------------------------------------------------------------------------- |
| Plan              | A project or learning goal container — type: `development` or `learning`                                         |
| Checklist Item    | A task within a plan, scoped as `daily` (recurring) or `longterm` (milestone)                                    |
| Daily Reset       | Automatic nightly reset of completed daily-scoped items to `done=false`                                          |
| Focus             | A plan's stated objective — fed to AI for context-aware suggestions                                              |
| Scope             | Whether a checklist item is `daily` (short-term, repeatable) or `longterm` (milestone-based)                     |
| Insights          | AI-generated suggestions — checklist item suggestions, daily suggestions, video recommendations                  |
| Resource          | A learning material (video, article, GitHub repo) linked to a plan                                               |
| Discovery         | The system that crawls YouTube to find relevant tutorial videos based on plan focus                              |
| Content Generator | Interface wrapping OpenAI GPT-4o for generating suggestions and search terms                                     |
| Calendar Entry    | A scheduled slot on the Smart Calendar — links a checklist item or recommendation to a specific day and position |
| Pinned Entry      | A calendar entry manually placed by the user — algorithm skips it during re-optimization                         |
| Slot Allocation   | The weighted distribution of daily/longterm/recommendation items per day                                         |

---

## Features (Implemented)

> These features are already built. Listed here for context — do not reimplement.

### Authentication & Users

- [x] User registration and login with email/password (JWT access + refresh tokens)
- [x] Password hashing with bcrypt
- [x] Seed data: test user and test plan for development

### Plans

- [x] CRUD for plans (create, read, update, delete)
- [x] Plan types: `development` or `learning`
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

### Smart Calendar

- [ ] Month-view calendar per plan with priority-ordered daily task lists
- [ ] Three content tiers: daily items (highest), longterm items (medium), AI recommendations (lowest)
- [ ] Flex ratios based on plan type: development (5/2/1), learning (3/2/3)
- [ ] Hard cap of 8 items per day, overflow pushes to next available day
- [ ] Cascade rule: unfilled higher-tier slots overflow to lower tiers
- [ ] Hybrid optimization: deterministic Go algorithm distributes, LLM ranks longterm urgency
- [ ] Triggers: page load, manual "Generate Schedule" button, reactive on task add/complete
- [ ] Full drag-and-drop: reorder within day, move between days
- [ ] Pinning: manually moved items marked `pinned=true`, skipped on re-optimization
- [ ] Reset: unpin all entries and re-generate schedule

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
- [ ] AI-generated daily focus narrative and reflection prompts
- [ ] User-configurable ratios and daily cap
- [ ] Google Calendar / external calendar sync
- [ ] Workspace / music integration
- [ ] Real-time notifications for scheduled items

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
├── plan_type (TEXT, NOT NULL — 'development' | 'learning')
├── daily_reset (BOOLEAN, NOT NULL, DEFAULT true)
├── created_at (TIMESTAMPTZ)
└── updated_at (TIMESTAMPTZ, trigger-updated)
```

### Checklist Items

```
checklist_items
├── id (PK, UUID, auto-generated)
├── plan_id (FK → plans, ON DELETE CASCADE)
├── description (TEXT, NOT NULL)
├── done (BOOLEAN, NOT NULL, DEFAULT false)
├── sequence (INTEGER, NOT NULL)
├── scope (TEXT, NOT NULL, DEFAULT 'longterm' — CHECK IN ('daily', 'longterm'))
├── scheduled_time (TIMESTAMPTZ, nullable)
├── archived (BOOLEAN, DEFAULT false)
├── created_at (TIMESTAMPTZ)
└── updated_at (TIMESTAMPTZ, trigger-updated)
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

### Calendar Entries (NEW — Smart Calendar)

```
calendar_entries
├── id (PK, UUID, auto-generated)
├── plan_id (FK → plans, ON DELETE CASCADE)
├── entry_date (DATE, NOT NULL)
├── position (INTEGER, NOT NULL — order within day, 1-8)
├── entry_type (TEXT, NOT NULL — CHECK IN ('daily', 'longterm', 'recommendation'))
├── checklist_item_id (FK → checklist_items, ON DELETE CASCADE, nullable)
├── recommendation_text (TEXT, nullable — for AI-generated recommendations)
├── recommendation_url (TEXT, nullable — optional link for video recs)
├── pinned (BOOLEAN, NOT NULL, DEFAULT false)
├── created_at (TIMESTAMPTZ)
└── updated_at (TIMESTAMPTZ, trigger-updated)

Indexes:
  UNIQUE (plan_id, entry_date, position) — one entry per slot per day per plan
  (plan_id, entry_date) — fast month-range lookups
  (plan_id, pinned) WHERE pinned = true — partial index for optimizer
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

| Method | Endpoint                                           | Auth | Description                         |
| ------ | -------------------------------------------------- | ---- | ----------------------------------- |
| GET    | `/api/plans/:id/checklists`                        | JWT  | List items (?scope=daily\|longterm) |
| GET    | `/api/plans/:id/checklists/archived`               | JWT  | List archived items                 |
| GET    | `/api/plans/:id/checklists/upcoming`               | JWT  | List upcoming scheduled items       |
| GET    | `/api/plans/:id/checklists/:checklist_id`          | JWT  | Get single item                     |
| POST   | `/api/plans/:id/checklists`                        | JWT  | Create item                         |
| PATCH  | `/api/plans/:id/checklists/:checklist_id`          | JWT  | Update item                         |
| DELETE | `/api/plans/:id/checklists/:checklist_id`          | JWT  | Delete item                         |
| PATCH  | `/api/plans/:id/checklists/:checklist_id/schedule` | JWT  | Set scheduled time                  |
| PATCH  | `/api/plans/:id/checklists/:checklist_id/archive`  | JWT  | Archive item                        |

### Insights API

| Method | Endpoint                                             | Auth | Description                      |
| ------ | ---------------------------------------------------- | ---- | -------------------------------- |
| GET    | `/api/insights/checklist-suggestion?plan_id=X`       | JWT  | AI suggests next task            |
| GET    | `/api/insights/checklist-suggestion-daily?plan_id=X` | JWT  | AI suggests 3 daily tasks        |
| GET    | `/api/insights/suggest-videos?plan_id=X`             | JWT  | AI finds relevant YouTube videos |

### Calendar API (NEW — Smart Calendar)

| Method | Endpoint                                 | Auth | Description                                  |
| ------ | ---------------------------------------- | ---- | -------------------------------------------- |
| GET    | `/api/plans/:id/calendar?month=2026-03`  | JWT  | Get calendar entries for month               |
| POST   | `/api/plans/:id/calendar/generate`       | JWT  | Trigger full schedule generation             |
| PATCH  | `/api/plans/:id/calendar/:entry_id/move` | JWT  | Move entry to day/position, sets pinned=true |
| PATCH  | `/api/plans/:id/calendar/:entry_id/pin`  | JWT  | Toggle pin status                            |
| POST   | `/api/plans/:id/calendar/reset`          | JWT  | Unpin all + re-optimize                      |

---

## Smart Calendar — Detailed Design

### Research Foundation

Slot allocation ratios grounded in three established frameworks:

- **70-20-10 Learning Model** (McCall, Lombardo, Eichinger): 70% learning from hands-on experience, 20% from relationships, 10% from formal education. Daily execution is the primary vehicle for progress.
- **1-3-5 Rule**: 1 big, 3 medium, 5 small tasks/day (9 total). Validates 8-item cap as reasonable cognitive load.
- **Eisenhower Matrix**: Urgent+Important first, then Important-Not-Urgent. Deliberate protection of Q2 time for sustained strategic growth.

### Slot Allocation (8 items/day hard cap)

| Plan Type     | Daily | Longterm | AI Recs | Rationale                                           |
| ------------- | ----- | -------- | ------- | --------------------------------------------------- |
| `development` | 5     | 2        | 1       | Execution-heavy — ship code, complete tasks         |
| `learning`    | 3     | 2        | 3       | Content-heavy — videos, articles alongside practice |

Cascade: unfilled higher-tier slots overflow downward. Never upward.

### Optimization Algorithm

**Hybrid approach:**

1. **Deterministic Go algorithm** — distributes items across days, respects caps, balances longterm workload, skips pinned items
2. **LLM ranking** (OpenAI via existing ContentGenerator interface) — ranks longterm items by urgency/relevance before distribution

**Pseudocode:**

```
func GenerateSchedule(plan, dailyItems, longtermItems, recommendations, pinnedEntries):
  1. Determine (dailyCap, longtermCap, recCap) from plan.PlanType
  2. Call LLM: rank longtermItems by urgency given plan.Focus
  3. For each day in month:
     a. Count pinned items → subtract from cap (8)
     b. Fill daily slots from unscheduled daily items
     c. Fill longterm slots from LLM-ranked items (spread evenly across days)
     d. Fill recommendation slots from AI-generated suggestions
     e. Cascade: unused higher-tier slots overflow to next tier
  4. Batch INSERT non-pinned entries
```

### Move/Pin Behavior

- PATCH move: updates entry_date + position, sets `pinned=true`
- Pinned entries survive re-optimization (algorithm skips them)
- POST reset: sets all entries for plan to `pinned=false`, then re-runs generate

---

## Business Rules

### Daily Reset

- Runs at 14:00 UTC daily via cron job
- Only resets items where `scope='daily'` AND `done=true` AND plan has `daily_reset=true`
- Uses bulk CTE update (not row-by-row)

### Checklist Scope Validation

- Scope must be `daily` or `longterm` — enforced by DB CHECK constraint and service-layer validation

### Calendar Slot Enforcement

- UNIQUE constraint `(plan_id, entry_date, position)` prevents double-booking
- Move endpoint returns 409 if target slot is occupied
- Hard cap of 8 positions per day enforced by algorithm (not DB constraint)

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

### Calendar Overflow

- If a day has 8 pinned items, no algorithm-generated items are placed there
- Overflow items push to next available day with remaining capacity
- If entire month is full, remaining items are not scheduled (frontend shows "unscheduled" bucket)

---

## Edge Cases

### Calendar Optimization

| Scenario                        | Handling                                                              |
| ------------------------------- | --------------------------------------------------------------------- |
| No checklist items exist        | Calendar is empty, only AI recommendations placed (if any)            |
| All items are pinned            | Algorithm generates nothing, calendar stays as-is                     |
| Task completed after scheduling | Entry remains on calendar (done status shown in UI), not auto-removed |
| Task deleted after scheduling   | CASCADE delete removes the calendar_entry                             |
| Plan type changed               | Next generate/reset recalculates with new ratios                      |
| LLM ranking fails               | Fallback to sequence-order for longterm items, log error              |

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

| Scenario                                    | Handling                                                    |
| ------------------------------------------- | ----------------------------------------------------------- |
| Multiple toasts at same time                | Stack vertically, each auto-dismisses independently         |
| localStorage cleared (hasSeenSidebarHint)   | Sidebar hint shows again on next plan page visit            |
| User opens sidebar then doesn't for 24hrs   | Reminder toast shown on next plan page visit (bottom-left)  |
| User never visits a plan page               | Sidebar hints never fire (only triggered on plan pages)     |
| Toast fired while another is dismissing     | New toast stacks, dismissing toast animates out             |

### Authorization & Ownership

| Scenario                                | Handling                                          |
| --------------------------------------- | ------------------------------------------------- |
| No Authorization header                 | 401 Unauthorized                                  |
| Expired JWT                             | 401 Unauthorized                                  |
| Valid JWT but plan belongs to other user | 403 Forbidden                                     |
| Checklist op on other user's plan       | 403 Forbidden (verify plan ownership first)       |
| Insights request for other user's plan  | 403 Forbidden                                     |
| Unauthenticated user visits `/`         | See splash/landing page                           |
| Authenticated user visits `/`           | Redirect to dashboard                             |

### Concurrent Access

| Scenario                             | Handling                                              |
| ------------------------------------ | ----------------------------------------------------- |
| Two generate requests simultaneously | Last write wins (acceptable for single-user per plan) |
| Move during generate                 | Pinned entry survives; generate only touches unpinned |

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
├── calendar/                  — NEW: Smart Calendar (handler, service, repository, model, optimizer)
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
