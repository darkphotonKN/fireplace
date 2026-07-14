# calendar-service — Specification

<!-- migrating to thin format: one line per capability, → FS-NNNN pointers -->

> Scope: per-plan Gantt calendar read-model (aggregation over plan-service). Platform maps: ../../CLAUDE.md.

calendar-service renders a **Plan Calendar** window for a single plan. It aggregates checklist-item date data owned by **plan-service** and formats it for the frontend Gantt view. It owns no checklist data and performs no mutations.

## Domain Terms

| Term                | Meaning                                                                                                   |
| ------------------- | --------------------------------------------------------------------------------------------------------- |
| **Plan Calendar**   | Per-plan Gantt-style calendar showing a plan's checklist items positioned by their `start_date`/`due_date`. |
| **Gantt Bar**       | A horizontal bar spanning `start_date..due_date` inclusive (rendered when both dates are set and `start < due`). |
| **Single-Day Chip** | A one-cell marker rendered when only one date is set, or when `start_date == due_date`.                    |
| **Window**          | The `[start, end]` date range derived from `view` + `date` (a month or a Sun..Sat week).                   |
| **week / month view** | The two supported calendar layouts. `view` = `"week"` or `"month"`; empty defaults to `"month"`.         |

## Features

**Implemented**
- `GetCalendar` — resolve a window, assert plan ownership, fetch overlapping checklist items from plan-service, format for render.

**Future (scaffolded, not active)**
- `calendar_entries` slot-pinning: pin an item to a specific `(scheduled_date, position 1-8)` slot.
- Recommendations: `entry_type='recommendation'` rows carrying `rec_title` / `rec_url` / `rec_description`.

The `calendar_entries` table and its migration exist and are wired through DI, but no running code reads or writes them.

## Read Model & Window Resolution

Input: `(planID, userID, view, date)`.

1. `view` defaults to `"month"` when empty.
2. **Window resolution:**
   - `view=month`, `date=YYYY-MM` → window `first-of-month .. last-of-month` (e.g. `2026-03` → `2026-03-01..2026-03-31`). Invalid format → `ErrInvalidInput`.
   - `view=week`, `date=YYYY-MM-DD` → the **Sunday..Saturday** week containing `date` (offset by `Weekday()`, Sunday=0). Invalid format → `ErrInvalidInput`.
   - Any other `view` → `ErrInvalidInput`.
3. **Ownership assertion:** `plan-service.AssertPlanOwnership(planID, userID)` — failure short-circuits (NotFound / Forbidden / etc.).
4. **Item fetch:** `plan-service ChecklistService.ListItemsInDateWindow(planID, userID, windowStart, windowEnd)`.
5. **Format:** each item → `CalendarItem` with `start_date` / `due_date` as `"YYYY-MM-DD"` or `""` when null.

**Window-intersection rule:** an item appears if its effective range `[start_date ?? due_date, due_date ?? start_date]` overlaps the window. This intersection filtering is performed **inside plan-service** (`ListItemsInDateWindow`), not here — calendar-service passes the resolved `[windowStart, windowEnd]` and formats whatever comes back.

**Archived items are excluded** from the response regardless of dates (enforced by plan-service's window query).

## Render Rules

The date state → render mapping (rendered by the FE from the returned `start_date`/`due_date`):

| State                              | Calendar render                          |
| ---------------------------------- | ---------------------------------------- |
| Both null                          | Not shown on calendar                    |
| Only `start_date` set              | Single-day chip on `start_date`          |
| Only `due_date` set                | Single-day chip on `due_date`            |
| Both set, `start_date == due_date` | Single-day chip on that date             |
| Both set, `start_date < due_date`  | Gantt bar spanning both dates inclusive  |
| Both set, `start_date > due_date`  | Invalid — rejected at write time (plan-service, 400) |

Grid rendering:
- **Month view:** 7-column grid for the requested month; bars span horizontally across day cells.
- **Week view:** 7-column grid for the Sun..Sat week containing `date`.
- Items appear only when at least one of `start_date` / `due_date` is set.
- Multiple items on the same day stack vertically within the cell.
- **Edge clipping:** a bar starting before the window renders clipped to the window's left edge; a bar ending after the window renders clipped to the right edge.

## gRPC Surface

Service `calendar.CalendarService` (`common/api/proto/calendar/calendar.proto`), port **:7104**.

`GetCalendar(GetCalendarRequest) → GetCalendarResponse`

**GetCalendarRequest**

| Field     | Type   | Notes                                            |
| --------- | ------ | ------------------------------------------------ |
| `plan_id` | string | UUID; malformed → InvalidArgument-class error    |
| `user_id` | string | UUID; malformed → InvalidArgument-class error    |
| `view`    | string | `"week"` or `"month"`; empty → `"month"`         |
| `date`    | string | `"YYYY-MM-DD"` for week, `"YYYY-MM"` for month    |

**GetCalendarResponse**

| Field          | Type                | Notes                        |
| -------------- | ------------------- | ---------------------------- |
| `plan_id`      | string              | echoed                       |
| `view`         | string              | resolved view                |
| `window_start` | string              | `"YYYY-MM-DD"`               |
| `window_end`   | string              | `"YYYY-MM-DD"`               |
| `items`        | repeated CalendarItem |                            |

**CalendarItem**: `id`, `description`, `scope`, `done`, `start_date` (`"YYYY-MM-DD"` or `""`), `due_date` (`"YYYY-MM-DD"` or `""`).

Errors: plan-service statuses are translated to domain sentinels (`NotFound→ErrNotFound`, `PermissionDenied→ErrForbidden`, `InvalidArgument→ErrInvalidInput`, `Unauthenticated→ErrUnauthorized`) and re-wrapped via `grpcerror.Fail`, preserving the correct code for the gateway to map to HTTP.

## Data Model (future / unused)

Table **`calendar_entries`** (DB `fireplace_calendar_service_db`, :5304) — created on boot, **not used by running code**:

| Column              | Type    | Notes                                             |
| ------------------- | ------- | ------------------------------------------------- |
| `id`                | UUID    | PK, `gen_random_uuid()`                           |
| `plan_id`           | UUID    | not null                                          |
| `checklist_item_id` | UUID    | nullable (plan-service owns checklist_items now)  |
| `entry_type`        | TEXT    | CHECK in (`daily`, `longterm`, `recommendation`)  |
| `scheduled_date`    | DATE    | not null                                          |
| `position`          | INTEGER | CHECK 1..8 (slot per day)                         |
| `pinned`            | BOOLEAN | default false                                     |
| `rec_title`         | TEXT    | recommendation payload                            |
| `rec_url`           | TEXT    | recommendation payload                            |
| `rec_description`   | TEXT    | recommendation payload                            |
| `created_at` / `updated_at` | TIMESTAMPTZ | `updated_at` maintained by trigger      |

Indexes: unique `(plan_id, scheduled_date, position)`; `(plan_id, scheduled_date)`; `(checklist_item_id)`. The migration is self-contained (carries its own `update_updated_at_column` trigger function and drops the old cross-table FKs from the monolith era).

## Edge Cases

| Scenario                                        | Handling                                                       |
| ----------------------------------------------- | -------------------------------------------------------------- |
| No checklist items have dates                   | Empty grid (empty `items`)                                     |
| Item has only `start_date` (or only `due_date`) | Single-day chip on the set date                                |
| `start_date == due_date`                        | Single-day chip                                                |
| Bar starts before window, ends inside           | Bar clipped to left edge of window                             |
| Bar starts inside window, ends after            | Bar clipped to right edge of window                            |
| Item archived                                   | Excluded from response regardless of dates                     |
| Item deleted                                    | Disappears (no separate calendar row)                          |
| Malformed `plan_id` / `user_id`                 | InvalidArgument-class error via `grpcerror.Fail`               |
| Invalid `view` / `date` format                  | `ErrInvalidInput`                                              |
| plan-service unavailable                        | Request fails (no local fallback / cache)                      |

## Owned elsewhere (cross-references)

- **Date data model** (`start_date` / `due_date` nullable DATE columns on `checklist_items`, `start_date <= due_date` CHECK) → **plan-service**.
- **Date mutation** `PATCH /api/plans/:id/checklists/:checklist_id/dates` (UpdateItemDates), including the `start_date > due_date` → 400 rejection → **plan-service**.
- **Window intersection + archived exclusion query** (`ListItemsInDateWindow`) → **plan-service**.
- **HTTP endpoint** `GET /api/plans/:id/calendar?view=&date=` → **api-gateway** (proxies to this service's gRPC).
- **Drag/drop UI** and drag-gesture → PATCH semantics → **client** (each gesture commits one `PATCH /dates`).
