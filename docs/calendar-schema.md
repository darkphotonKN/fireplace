# Plan Calendar — Schema & Architecture

The Plan Calendar is a per-plan Gantt-style calendar. Checklist items with at least one of `start_date` or `due_date` set appear on the calendar; date ranges render as horizontal bars, single dates render as one-day chips.

## 1. Schema Changes

### checklist_items (modified)

| Column         | Type        | Notes                                                                  |
| -------------- | ----------- | ---------------------------------------------------------------------- |
| start_date     | DATE        | nullable — Gantt range start                                            |
| due_date       | DATE        | nullable — Gantt range end                                              |
| ~~scheduled_time~~ | ~~TIMESTAMPTZ~~ | **Removed.** Date portion backfilled into `start_date` before drop. |

**CHECK constraint:** `start_date IS NULL OR due_date IS NULL OR start_date <= due_date`

**Indexes:**

| Name                          | Columns                       | Purpose                              |
| ----------------------------- | ----------------------------- | ------------------------------------ |
| idx_checklist_plan_start_date | `(plan_id, start_date)`       | Window queries — start anchored      |
| idx_checklist_plan_due_date   | `(plan_id, due_date)`         | Window queries — due anchored        |

### calendar_entries (dropped)

The slot-based `calendar_entries` table is dropped. The Plan Calendar reads directly from `checklist_items` — no separate calendar storage, no slot/position model, no pinning.

## 2. Migration

Sequenced as a single migration to remain reversible:

```sql
-- up
ALTER TABLE checklist_items
  ADD COLUMN start_date DATE,
  ADD COLUMN due_date   DATE;

UPDATE checklist_items
SET start_date = scheduled_time::date
WHERE scheduled_time IS NOT NULL;

ALTER TABLE checklist_items
  ADD CONSTRAINT chk_checklist_dates
  CHECK (start_date IS NULL OR due_date IS NULL OR start_date <= due_date);

ALTER TABLE checklist_items DROP COLUMN scheduled_time;

CREATE INDEX idx_checklist_plan_start_date ON checklist_items (plan_id, start_date);
CREATE INDEX idx_checklist_plan_due_date   ON checklist_items (plan_id, due_date);

DROP TABLE IF EXISTS calendar_entries;
```

```sql
-- down
-- (recreate calendar_entries from migration 000013, restore scheduled_time)
ALTER TABLE checklist_items ADD COLUMN scheduled_time TIMESTAMPTZ;
UPDATE checklist_items SET scheduled_time = start_date::timestamptz WHERE start_date IS NOT NULL;
ALTER TABLE checklist_items DROP CONSTRAINT chk_checklist_dates;
ALTER TABLE checklist_items DROP COLUMN start_date, DROP COLUMN due_date;
DROP INDEX IF EXISTS idx_checklist_plan_start_date;
DROP INDEX IF EXISTS idx_checklist_plan_due_date;
-- recreate calendar_entries (see 000013_create_calendar_entries_table.up.sql)
```

## 3. API

### GET `/api/plans/:id/calendar?view=<week|month>&date=<date>`

Returns checklist items in the requested window.

| Param   | Format                  | Notes                                                |
| ------- | ----------------------- | ---------------------------------------------------- |
| view    | `week` \| `month`       | Defaults to `month` if omitted                       |
| date    | `YYYY-MM` or `YYYY-MM-DD` | For `month`, accepts `YYYY-MM`; for `week`, the week containing this date |

**Window resolution:**

- `view=month`, `date=2026-03` → `2026-03-01..2026-03-31`
- `view=week`, `date=2026-03-09` → Sun..Sat week containing 2026-03-09

**Filter:** items where `[COALESCE(start_date, due_date), COALESCE(due_date, start_date)]` overlaps the window AND `archived = false`.

**Response shape:**

```json
{
  "statusCode": 200,
  "message": "Calendar entries retrieved successfully",
  "result": {
    "planId": "a1b2c3d4-...",
    "view": "month",
    "windowStart": "2026-03-01",
    "windowEnd": "2026-03-31",
    "items": [
      {
        "id": "c1d2e3f4-...",
        "description": "Build auth middleware",
        "scope": "longterm",
        "done": false,
        "startDate": "2026-03-04",
        "dueDate":   "2026-03-12"
      },
      {
        "id": "d4e5f6g7-...",
        "description": "Standup",
        "scope": "daily",
        "done": false,
        "startDate": "2026-03-10",
        "dueDate":   null
      }
    ]
  }
}
```

### PATCH `/api/plans/:id/checklists/:checklist_id/dates`

Updates `start_date` and/or `due_date`. Both fields are optional in the body; absent keys leave existing values untouched. Either may be `null` to clear.

**Request:**

```json
{ "startDate": "2026-03-10", "dueDate": "2026-03-14" }
```

**Validation:**

- If both resolved values (post-merge with current DB row) are non-null, enforce `start_date <= due_date`
- 400 on violation; neither field is persisted

**Response:** the updated checklist item (same shape as `GET /checklists/:id`).

### Removed

- `PATCH /api/plans/:id/checklists/:checklist_id/schedule` — replaced by `/dates`
- `POST /api/plans/:id/calendar/generate`, `PATCH .../move`, `PATCH .../pin`, `POST .../reset` — slot/AI scheduling out of scope

## 4. Package Layout

```
internal/calendar/
├── handler.go     GET window handler
├── service.go     Window resolution, ownership check, repo call
├── repository.go  SELECT items where date range overlaps window
└── model.go       Request/response DTOs (GetCalendarResp, CalendarItem)

internal/checklistitems/
├── handler.go     UpdateDates handler (new)
├── service.go     UpdateDates with cross-field validation (new)
├── repository.go  UpdateDates query (new)
└── model.go       UpdateDatesReq DTO (new)
```

The pure `scheduler.go` from the prior Smart Calendar design is removed (no slot algorithm).

## 5. Drag-and-Drop Behavior (Frontend)

| Gesture                        | dates change                                          |
| ------------------------------ | ----------------------------------------------------- |
| Drag bar middle by Δ days      | `start_date += Δ`, `due_date += Δ`                    |
| Drag bar left edge by Δ days   | `start_date += Δ`                                     |
| Drag bar right edge by Δ days  | `due_date += Δ`                                       |
| Drag chip by Δ days            | Whichever date is set shifts by Δ; if both equal, both shift |

Each gesture commits via a single `PATCH /dates` request; FE reverts on 400.
