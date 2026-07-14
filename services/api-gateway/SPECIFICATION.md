# api-gateway — Specification

<!-- migrating to thin format: one line per capability, → FS-NNNN pointers -->

> Scope: HTTP edge/BFF — API surface, edge auth, and the gateway-owned Notes & User Analytics domains. Platform maps: ../../CLAUDE.md.

## Role

The api-gateway is the **only HTTP surface** in the Fireplace platform — the edge/BFF the frontend client talks to. It terminates HTTP (Gin), authenticates every protected request at the edge with the shared JWT, and proxies to the gRPC domain services (auth, plan, calendar) discovered via Consul. Two domains still live *inside* the gateway and are owned here: **Notes** and **User Analytics**. Everything else it exposes is a routing entry to a downstream service (see cross-refs at the bottom).

## API Surface

Full REST catalog exposed by the gateway. All routes are under base path `/api`. Auth = `JWT` means the route is behind the edge auth middleware. Insights rows are **routing-only** — their behavior is specified in insights-service.

| Method | Endpoint | Auth | Routes to | Notes |
|---|---|---|---|---|
| POST | `/api/users/signup` | public | auth-service | Create user |
| POST | `/api/users/signin` | public | auth-service | Login → access/refresh tokens |
| GET | `/api/users/profile` | JWT | auth-service | Current user's profile (`sub` claim) |
| PATCH | `/api/users/profile` | JWT | auth-service | Update name/displayName/bio |
| GET | `/api/users/:id` | JWT | auth-service | User by id |
| GET | `/api/users` | JWT | auth-service | List users |
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

- **Users, JWT issuance, refresh tokens** → auth-service (gateway only *validates* tokens at the edge).
- **Plans, checklist items, daily-reset logic, Gantt/search data** → plan-service.
- **Calendar read-model** → calendar-service.
- **AI insights / video suggestions** → insights-service (the `/api/insights/*` routes and leftover `internal/insights`, `internal/discovery`, `internal/ai` code here are strangler cleanup, not a gateway feature).
- **Landing page + all UI** → client (React/Next.js).
