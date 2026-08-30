# api-gateway — Specification

<!-- migrating to thin format: one line per capability, → FS-NNNN pointers -->

> Scope: HTTP edge/BFF — API surface, edge auth, and the gateway-owned Notes & User Analytics domains. Platform maps: ../../CLAUDE.md.

## Role

The api-gateway is the **only HTTP surface** in the Fireplace platform — the edge/BFF the frontend client talks to. It terminates HTTP (Gin), authenticates every protected request at the edge with the shared JWT, and proxies to the gRPC domain services (plan, insights) discovered via Consul. Domains owned *inside* the gateway: **Users & Identity**, **Notes**, and **User Analytics** — the first of these folded back in from auth-service under ADR-0009 §1. Everything else it exposes is a routing entry to a downstream service (see cross-refs at the bottom).

## Users

- [x] Profile view and edit → FS-none
- [x] Typed (serialized) profile surface → FS-0002
- [x] Registration and login with email/password → FS-none
- [x] Refresh-token exchange for a new token pair → FS-none
- [x] User read and list → FS-none
- [x] `user.created` / `user.deleted` published on `auth.events` → FS-none
- [ ] `user.updated` event → FS-none
- [ ] Password change / reset flow → FS-none
- [ ] Refresh-token revocation → FS-none

## Contract

- [x] Legacy enveloped REST surface → FS-none
- [ ] Whole gateway surface serialized and governed → FS-0004

## Authorization

- [x] Edge authentication on every protected route → FS-none
- [ ] Per-user authorization on plan-scoped resources → FS-0005

<!--
LEGACY BELOW: every section from here down predates the thin format and still carries
FS-level detail inline. Per migrate-only-what-you-touch, they are left as found rather
than silently rewritten — flagged for a dedicated hygiene pass.
-->

## API Surface

Full REST catalog exposed by the gateway. All routes are under base path `/api`. Auth = `JWT` means the route is behind the edge auth middleware. Insights rows are **routing-only** — their behavior is specified in insights-service.

| Method | Endpoint | Auth | Routes to | Notes |
|---|---|---|---|---|
| POST | `/api/users/signup` | public | **gateway-local** | Create user |
| POST | `/api/users/signin` | public | **gateway-local** | Login → access/refresh tokens |
| GET | `/api/users/profile` | JWT | **gateway-local** | Current user's profile (`sub` claim) |
| PATCH | `/api/users/profile` | JWT | **gateway-local** | Update name/displayName/bio |
| GET | `/api/users/:id` | JWT | **gateway-local** | User by id |
| GET | `/api/users` | JWT | **gateway-local** | List users |
| GET | `/api/plans` | JWT | plan-service | List caller's plans |
| GET | `/api/plans/search` | JWT | plan-service | Search plans |
| GET | `/api/plans/shared` | JWT | plan-service | Plans shared with caller |
| GET | `/api/plans/:id` | JWT | plan-service | Plan by id |
| POST | `/api/plans` | JWT | plan-service | Create plan |
| PATCH | `/api/plans/:id` | JWT | plan-service | Update plan |
| PATCH | `/api/plans/:id/toggle-daily-reset` | JWT | plan-service | Toggle daily-reset flag |
| DELETE | `/api/plans/:id` | JWT | plan-service | Delete plan |
| GET | `/api/plans/:id/checklists` | JWT | plan-service | List checklist items |
| GET | `/api/plans/:id/checklists/archived` | JWT | plan-service | Archived items |
| GET | `/api/plans/:id/checklists/upcoming` | JWT | plan-service | Upcoming (scheduled) items |
| GET | `/api/plans/:id/checklists/:checklist_id` | JWT | plan-service | Item by id |
| POST | `/api/plans/:id/checklists` | JWT | plan-service | Create item |
| PATCH | `/api/plans/:id/checklists/:checklist_id` | JWT | plan-service | Update item |
| DELETE | `/api/plans/:id/checklists/:checklist_id` | JWT | plan-service | Delete item |
| PATCH | `/api/plans/:id/checklists/:checklist_id/dates` | JWT | plan-service | Update start/due dates |
| PATCH | `/api/plans/:id/checklists/:checklist_id/archive` | JWT | plan-service | Archive item |
| GET | `/api/plans/:id/calendar` | JWT | calendar-service | Calendar read-model (`?view=week\|month&date=`) |
| GET | `/api/insights/checklist-suggestion` | JWT | insights domain | routing only; spec in insights-service |
| GET | `/api/insights/checklist-suggestion-daily` | JWT | insights domain | routing only; spec in insights-service |
| GET | `/api/insights/suggest-videos` | JWT | insights domain | routing only; spec in insights-service |
| GET | `/api/plans/:id/notes` | JWT | **gateway-local** | List notes (filterable) |
| GET | `/api/plans/:id/notes/:noteId` | JWT | **gateway-local** | Note by id |
| POST | `/api/plans/:id/notes` | JWT | **gateway-local** | Create note |
| PATCH | `/api/plans/:id/notes/:noteId` | JWT | **gateway-local** | Update note |
| DELETE | `/api/plans/:id/notes/:noteId` | JWT | **gateway-local** | Delete note |
| POST | `/api/plans/:id/notes/generate-ai` | JWT | **gateway-local** | AI-generated notes |
| GET | `/api/analytics/user/:userId` | JWT | **gateway-local** | Per-user daily analytics |
| GET | `/swagger/*any` | public | — | Swagger UI (OpenAPI 2.0) |

## Edge Authentication & Authorization

*(Backend only. The landing page and all UI live in the client — see cross-refs.)*

- **Token validation at the edge.** `internal/auth.AuthMiddleware` runs on the protected route group. It reads the `Authorization: Bearer <token>` header, parses/validates the JWT locally against the shared `JWT_SECRET` (no remote call to auth-service, which issues tokens with the same secret), and extracts the user id from the `sub` claim into `gin.Context`. `GetUserID` reads it back for handlers.
- **401 Unauthorized** on: missing header, malformed header, invalid/expired token, or an unparseable user id. The response is a generic `{"statusCode":401,"message":"unauthorized"}`; the specific reason is logged server-side only.
- **Ownership / per-user filtering** is enforced by passing the authenticated `userID` down to the domain services on writes and owner-scoped reads (e.g. plan create/update, calendar). Plan-service and calendar-service perform the actual **403 Forbidden** ownership checks (e.g. via `AssertPlanOwnership`); the gateway supplies the caller identity and never trusts a client-supplied user id.

## Owned Feature: Users & Identity

Folded back in from auth-service under ADR-0009 §1 — user identity, credentials, JWT
issuance, and profiles now run in-process (`internal/gateway/auth`). The gateway both
**issues** tokens here and **validates** them at the edge; verification itself stays shared
in `common/auth` (ADR-0009 §5) because plan-service and insights-service verify
independently. Do not duplicate the verifier into this package.

**Domain terms.** **User** — an account: `id`, `name`, `email` (unique), bcrypt-hashed
password, optional `display_name` + `bio`. **Access token** — short-lived JWT (type
`access`). **Refresh token** — medium-lived JWT (type `refresh`), stateless, no server
store. **`display_name`** — user-chosen presentation name, falling back to `name` in UI.

**Business rules.**

- **JWT lifecycle.** Sign-up, sign-in and refresh each issue an `access` + `refresh` pair
  signed with the shared `JWT_SECRET`. TTLs come from env (`ACCESS_TOKEN_TTL` /
  `REFRESH_TOKEN_TTL`); the deployed `.env` sets access 24h, refresh 168h. Code-const
  fallbacks when unset or unparseable: access 1h, refresh 168h.
- **bcrypt.** Passwords hashed at `bcrypt.DefaultCost` on sign-up, constant-time compared
  on sign-in. The hash never leaves the process (JSON `-`; the list query blanks it).
- **Account-enumeration-safe sign-in.** An unknown email and a wrong password return the
  same `Unauthorized` result — the difference is never exposed.
- **Cascade via events.** Deleting a user emits `user.deleted`; plan-service and
  insights-service consume it to clean up their own data. This domain never reaches into
  another service's tables.

**Events.** Exchange `auth.events` (topic, durable); protobuf, persistent. `user.created`
on sign-up, `user.deleted` on delete, `user.updated` a logged stub not yet published. The
gateway became an event producer when this folded in — the exchange is declared in
`cmd/main.go`. **Consumed:** none; auth-service's no-op consumer stub was dropped with the
fold rather than carried over.

### `users` (migration 000023)

Recreated in the gateway DB by `000023_recreate_users_from_auth_service`. The gateway
created this table in 000001 and dropped it in 000020 when auth-service was extracted;
000023 is a forward recreation matching auth-service's schema as it stood at the fold, not
a revert. The FK constraints 000020 dropped are **not** restored — plans live in
plan-service's database, so `user_id` columns stay plain UUIDs with integrity enforced at
the application/event layer.

| Column | Type | Constraints |
|---|---|---|
| `id` | UUID | PK, default `uuid_generate_v4()` |
| `name` | TEXT | NOT NULL |
| `email` | TEXT | UNIQUE, NOT NULL |
| `password` | TEXT | NOT NULL (bcrypt hash; Go field `HashedPassword`) |
| `display_name` | TEXT | nullable |
| `bio` | TEXT | nullable |
| `created_at` / `updated_at` | TIMESTAMP | default `CURRENT_TIMESTAMP` |

No `refresh_tokens` table — refresh tokens are stateless JWTs.

**Edge cases.**

| Scenario | Behavior |
|---|---|
| Sign up with existing email | Unique-constraint violation → domain conflict error |
| Sign up missing name/email/password | `ErrInvalidInput` |
| Sign in, unknown email | `Unauthorized` (indistinguishable from wrong password) |
| Sign in, wrong password | `Unauthorized` |
| Refresh with expired/invalid token | `Unauthorized` |
| `UpdateProfile` with empty `name` | `ErrInvalidInput` |
| `UpdateProfile` with no fields | No-op update; returns current user |
| `GetUser` / `DeleteUser` unknown id | `NotFound` |
| Malformed UUID | `InvalidArgument` |

> Not in scope: avatar, timezone, notifications, public profiles.

## Owned Feature: Notes

⏳ Strangler: currently owned by the gateway; a candidate for future extraction.

Notes are free-form and AI-assisted annotations attached to a plan. The gateway owns the `notes` table and the full CRUD + AI-generation logic (`internal/notes`).

- **CRUD**: create, read (single + list-by-plan with filters by type/priority/tags/isRead/isDismissed/relatedTaskId), update (partial), delete. On create, a plan-existence check is done via the plan adapter (gRPC to plan-service); tags are auto-derived from content when none are supplied.
- **Types**: `user`, `ai`, `warning`, `insight`, `suggestion`. **Priorities**: `low`, `medium`, `high`, `critical` (defaults: type `user`, priority `medium`).
- **AI notes** (`POST .../notes/generate-ai`): pulls the plan + its daily checklist items (via the plan adapter), then generates one or more notes. `warning`/`insight`/`suggestion` requests each run a targeted generator; an empty/`all` request generates all three. The `suggestion` generator uses OpenAI when available and falls back to rule-based content; `warning`/`insight` are rule-based over task counts, overdue/completion analysis. Generated notes carry `ai_metadata` (source, confidence, context, timestamp).

## Owned Feature: User Analytics

⏳ Strangler: currently owned by the gateway; a candidate for future extraction.

Per-user daily productivity metrics (`internal/useranalytics`), backed by the `user_analytics` table (one row per user per day). Exposes `GET /api/analytics/user/:userId`. The service loads the user's checklist data via the plan adapter and reads the day's analytics row.

> Implementation note (ground truth): the repository (`GetByUserAndDate`) and HTTP handler are currently **stubs** — the route, table, and service wiring exist, but the read path returns `nil` / `501 Not Implemented`. The domain and ownership are the gateway's; the data path is not yet completed.

## Background Jobs

- **Nightly daily-reset trigger** (`internal/jobs/daily_reset_job.go` + `manager.go`): a cron job scheduled at `0 0 14 * * *` (14:00 UTC daily) calls the plan adapter's `ResetDailyItems`, which invokes plan-service's `ChecklistService.DailyReset` over gRPC. The gateway is only the **scheduler/trigger** — the reset *logic* (which daily items to reset and how) lives in plan-service.
- No other scheduled jobs are registered (the old scheduled-items/reminder job is not present here).

## Data Model

### `notes` (migration 000012)

| Column | Type | Notes |
|---|---|---|
| `id` | UUID PK | `gen_random_uuid()` |
| `plan_id` | UUID | plain UUID; FK dropped in migration 21 (integrity app-side) |
| `content` | TEXT | required |
| `type` | VARCHAR(20) | default `user` |
| `priority` | VARCHAR(20) | default `medium` |
| `tags` | TEXT[] | GIN-indexed |
| `related_task_ids` | UUID[] | GIN-indexed |
| `is_read` | BOOLEAN | default false |
| `is_dismissed` | BOOLEAN | default false |
| `ai_metadata` | JSONB | AI generation metadata |
| `created_at` / `updated_at` | TIMESTAMP | `updated_at` via trigger |

### `user_analytics` (migration 000011)

| Column | Type | Notes |
|---|---|---|
| `id` | UUID PK | `gen_random_uuid()` |
| `user_id` | UUID | plain UUID; FK dropped in migration 20 |
| `date` | DATE | not null; `UNIQUE(user_id, date)` |
| `tasks_completed` | INTEGER | default 0 |
| `tasks_total` | INTEGER | default 0 |
| `completion_rate` | DECIMAL(3,2) | 0.00–1.00 |
| `current_streak` | INTEGER | consecutive days with >0 completed |
| `active_plans_count` | INTEGER | default 0 |
| `created_at` / `updated_at` | TIMESTAMPTZ | `updated_at` via trigger |

## Owned elsewhere (cross-refs)

- **Plans, checklist items, daily-reset logic, Gantt/search data** → plan-service.
- **Calendar read-model** → calendar-service.
- **AI insights / video suggestions** → insights-service. The `/api/insights/*` routes remain here because the gateway is the only HTTP surface, but they now proxy to insights-service over gRPC via `internal/gateway/insights`. The in-process implementation (`internal/insights` service + repository, `internal/discovery`, `internal/concepts`, and the checklist/search-term generators) was **deleted** when the routes were repointed — the strangler cleanup is done. What remains under `internal/insights` is the gateway's own typed HTTP transport layer, and `internal/ai` now serves **Notes only**.
- **Landing page + all UI** → client (React/Next.js).
