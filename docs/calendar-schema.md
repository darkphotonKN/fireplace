# Smart Calendar — Schema & Architecture

## 1. Existing Tables (Referenced)

### plans

| Column | Type | Notes (Calendar relevance) |
|--------|------|---------------------------|
| id | UUID | PK, `gen_random_uuid()` |
| user_id | UUID | FK → users(id) ON DELETE CASCADE |
| name | TEXT | NOT NULL |
| description | TEXT | |
| plan_type | TEXT | NOT NULL — **drives ratio selection** (development → 5/2/1, learning → 3/2/3) |
| focus | TEXT | NOT NULL — **used as LLM context** for ranking longterm items by urgency |
| daily_reset | BOOLEAN | NOT NULL, default `true` |
| created_at | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| updated_at | TIMESTAMPTZ | NOT NULL, default `NOW()` |

### checklist_items

| Column | Type | Notes (Calendar relevance) |
|--------|------|---------------------------|
| id | UUID | PK, `gen_random_uuid()` |
| plan_id | UUID | FK → plans(id) ON DELETE CASCADE |
| description | TEXT | NOT NULL |
| done | BOOLEAN | NOT NULL, default `false` |
| sequence | INTEGER | NOT NULL |
| scope | TEXT | NOT NULL, CHECK `('longterm', 'daily')` — **maps to entry_type** (daily → daily, longterm → longterm) |
| scheduled_time | TIMESTAMPTZ | nullable — **pins daily items to specific dates** in the scheduler |
| archived | BOOLEAN | default `FALSE` — **archived items are excluded** from calendar scheduling |
| created_at | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| updated_at | TIMESTAMPTZ | NOT NULL, default `NOW()` |

### resources

| Column | Type | Notes (Calendar relevance) |
|--------|------|---------------------------|
| id | UUID | PK, `gen_random_uuid()` |
| plan_id | UUID | FK → plans(id) ON DELETE CASCADE |
| resource_type | TEXT | NOT NULL |
| url | TEXT | NOT NULL |
| title | TEXT | NOT NULL |
| description | TEXT | |
| sequence | INTEGER | NOT NULL |
| created_at | TIMESTAMPTZ | NOT NULL, default `NOW()` |
| updated_at | TIMESTAMPTZ | NOT NULL, default `NOW()` |

---

## 2. New Table: calendar_entries

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| id | UUID | PRIMARY KEY, default `gen_random_uuid()` | |
| plan_id | UUID | NOT NULL, FK → plans(id) ON DELETE CASCADE | Scopes entries to one plan |
| checklist_item_id | UUID | nullable, FK → checklist_items(id) ON DELETE SET NULL | Links daily/longterm entries to their source task. NULL for recommendations. |
| entry_type | TEXT | NOT NULL, CHECK `(entry_type IN ('daily', 'longterm', 'recommendation'))` | Determines tier for ratio budgeting |
| scheduled_date | DATE | NOT NULL | The calendar day this entry occupies |
| position | INTEGER | NOT NULL, CHECK `(position >= 1 AND position <= 8)` | Slot order within the day (1 = top) |
| pinned | BOOLEAN | NOT NULL, default `false` | `true` = manually placed, survives re-optimization |
| rec_title | TEXT | | Recommendation-only: display title |
| rec_url | TEXT | | Recommendation-only: link URL |
| rec_description | TEXT | | Recommendation-only: short summary |
| created_at | TIMESTAMPTZ | NOT NULL, default `NOW()` | |
| updated_at | TIMESTAMPTZ | NOT NULL, default `NOW()` | |

### Indexes

| Name | Columns | Type | Purpose |
|------|---------|------|---------|
| uq_calendar_plan_date_pos | `(plan_id, scheduled_date, position)` | UNIQUE | Enforces one entry per slot |
| idx_calendar_plan_date | `(plan_id, scheduled_date)` | B-tree | Fast month-range queries |
| idx_calendar_checklist_item | `(checklist_item_id)` | B-tree | Reactive updates on task changes |

### Constraints

- `entry_type` must be one of `'daily'`, `'longterm'`, `'recommendation'`
- `position` must be between 1 and 8 inclusive (hard daily cap)
- `checklist_item_id` is nullable (recommendations have no backing task)
- ON DELETE SET NULL for `checklist_item_id` — entry survives if task is deleted (becomes orphaned, cleaned on next optimization)
- ON DELETE CASCADE for `plan_id` — deleting a plan removes all its calendar entries

---

## 3. Package Structure

```
internal/calendar/
├── handler.go      HTTP handlers
├── service.go      Orchestration layer
├── repository.go   Database operations
├── model.go        Request/response structs
└── scheduler.go    Pure scheduling function

internal/constants/
└── calendar.go     Slot cap, ratios, entry type constants
```

### File Responsibilities

| File | Responsibility |
|------|---------------|
| `handler.go` | GetMonth, Generate, MoveEntry, ReorderEntry, Reset — parse requests, call service, format responses |
| `service.go` | Fetch checklist items + recommendations, call LLM for longterm ranking, invoke scheduler, persist results |
| `repository.go` | Batch upsert entries, delete unpinned by plan+month, get entries by date range, update single entry position/date/pin |
| `model.go` | `CalendarEntry`, `MonthResponse`, `DaySlot`, `MoveEntryReq`, `ReorderReq`, `GenerateReq` |
| `scheduler.go` | `ScheduleMonth()` — pure function, no DB, no side effects. Takes items + config, returns `[]CalendarEntry` |

---

## 4. Data Flow Diagram

```
                          GET /api/plans/:id/calendar?month=2026-03
                                        │
                                        ▼
                               ┌─────────────────┐
                               │   handler.go     │
                               │   GetMonth()     │
                               └────────┬─────────┘
                                        │
                                        ▼
                               ┌─────────────────┐
                               │   service.go     │
                               │   GetMonth()     │
                               └────────┬─────────┘
                                        │
                           ┌────────────┴────────────┐
                           ▼                         ▼
                  ┌─────────────────┐      ┌──────────────────┐
                  │ repo.GetByMonth │      │ Entries exist?   │
                  │ (plan, month)   │      │                  │
                  └────────┬────────┘      └────────┬─────────┘
                           │                        │
                           ▼                        │
                    ┌──────────────┐                │
                    │ Has entries?  │                │
                    └──┬───────┬───┘                │
                   YES │       │ NO                 │
                       ▼       ▼                    │
              ┌──────────┐  ┌──────────────────┐    │
              │ Return   │  │ Generate flow    │◄───┘
              │ entries  │  │ (same as POST    │
              └──────────┘  │  /generate)      │
                            └───────┬──────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
           ┌──────────────┐ ┌────────────┐ ┌───────────────┐
           │ Fetch daily  │ │ Fetch      │ │ Fetch         │
           │ & longterm   │ │ plan       │ │ recommendations│
           │ checklist    │ │ (type,     │ │ via insights  │
           │ items        │ │  focus)    │ │ service       │
           │ (not archived│ │            │ │               │
           └──────┬───────┘ └─────┬──────┘ └───────┬───────┘
                  │               │                 │
                  │               ▼                 │
                  │      ┌──────────────────┐       │
                  │      │ LLM: rank        │       │
                  │      │ longterm items   │       │
                  │      │ by urgency       │       │
                  │      │ (plan.focus as   │       │
                  │      │  context)        │       │
                  │      └────────┬─────────┘       │
                  │               │                 │
                  ▼               ▼                 ▼
              ┌─────────────────────────────────────────┐
              │          scheduler.go                    │
              │          ScheduleMonth()                 │
              │  (pure function — no DB, no side effects)│
              │                                         │
              │  Inputs: items, recommendations,        │
              │          pinned entries, plan_type,      │
              │          month range                     │
              │                                         │
              │  Output: []CalendarEntry                │
              └──────────────────┬──────────────────────┘
                                 │
                                 ▼
                    ┌────────────────────────┐
                    │ repo.BatchUpsert()     │
                    │ (delete unpinned first,│
                    │  insert new entries)   │
                    └────────────┬───────────┘
                                 │
                                 ▼
                    ┌────────────────────────┐
                    │ Return MonthResponse   │
                    │ (days[] with entries)  │
                    └────────────────────────┘


  POST /generate — same flow, always runs generate (skips "has entries?" check)
  POST /reset   — deletes ALL entries (including pinned), then runs generate
  PATCH /move   — repo.UpdateEntry(id, newDate, newPos, pinned=true)
  PATCH /reorder— repo.UpdateEntry(id, sameDate, newPos, pinned=true)
```

---

## 5. Scheduling Algorithm

### ScheduleMonth Pure Function

```
FUNCTION ScheduleMonth(
    dailyItems      []ChecklistItem,    // scope=daily, not archived
    longtermItems   []ChecklistItem,    // scope=longterm, not archived, LLM-ranked
    recommendations []Recommendation,   // from insights service
    pinnedEntries   []CalendarEntry,    // existing pinned entries to preserve
    planType        string,             // "development" or "learning"
    monthRange      (startDate, endDate)
) -> []CalendarEntry

1. INIT day slots
   For each day in monthRange:
       slots[day] = array of 8 nulls
       remaining[day] = 8

2. PRE-FILL pinned entries
   For each pinned entry:
       slots[entry.date][entry.position] = entry
       remaining[entry.date] -= 1

3. CALCULATE tier budgets per day
   For each day:
       capacity = remaining[day]
       ratios = getRatios(planType)
           development: {daily: 5, longterm: 2, rec: 1}
           learning:    {daily: 3, longterm: 2, rec: 3}
       total_ratio = sum(ratios)
       budget[day] = {
           daily:    floor(capacity * ratios.daily / total_ratio)
           longterm: floor(capacity * ratios.longterm / total_ratio)
           rec:      capacity - budget.daily - budget.longterm   // remainder
       }

4. PLACE daily items
   Sort by: scheduled_time ASC (non-null first), then sequence ASC
   For each daily item:
       IF item.scheduled_time is set AND falls within month:
           target_day = item.scheduled_time.date
       ELSE:
           target_day = first day with budget.daily > 0
       IF budget[target_day].daily > 0:
           place in next open slot on target_day
           budget[target_day].daily -= 1
       ELSE:
           // CASCADE: try longterm/rec budget on same day
           // OVERFLOW: try next day with any capacity
           forward-fill to next available day

5. PLACE longterm items (already ranked by LLM)
   Spread evenly across days, preferring days with most remaining longterm budget
   For each longterm item:
       target_day = day with highest longterm budget (ties: earliest day)
       IF budget[target_day].longterm > 0:
           place in next open slot
           budget[target_day].longterm -= 1
       ELSE:
           // CASCADE into rec budget, then OVERFLOW to next day

6. PLACE recommendations
   Fill remaining budget, spread for variety (no two same-type recs on same day if possible)
   For each recommendation:
       target_day = day with highest rec budget
       place in next open slot
       budget[target_day].rec -= 1

7. RETURN flat list of all CalendarEntry (pinned + newly placed)
```

---

## 6. Ratio Configuration

| Plan Type | Daily | Longterm | Recommendation | Total |
|-----------|-------|----------|----------------|-------|
| development | 5 | 2 | 1 | 8 |
| learning | 3 | 2 | 3 | 8 |

### Cascade Rule

When a higher-tier budget is unfilled (e.g., only 3 daily items exist but budget is 5), the 2 unused daily slots cascade downward:

1. First overflow to **longterm** budget
2. Then overflow to **recommendation** budget

This ensures all 8 slots per day are utilized regardless of item availability in each tier.

### Example (Development plan, day has 2 daily items available)

| Tier | Original Budget | Available Items | Placed | Cascade |
|------|----------------|-----------------|--------|---------|
| Daily | 5 | 2 | 2 | 3 slots → longterm |
| Longterm | 2 + 3 = 5 | 4 | 4 | 1 slot → rec |
| Recommendation | 1 + 1 = 2 | 10 | 2 | — |
| **Total** | **8** | | **8** | |

---

## 7. LLM Integration

### Pattern

Follows the same approach as `insights.AcquireGenRelevantData()` — gather context, build prompt, call `ai.Generator`, parse response.

### Ranking Flow

1. Fetch plan details (`plan.focus`, `plan.name`, `plan.description`)
2. Fetch all non-archived longterm checklist items for the plan
3. Build prompt:

```
Given the plan "{plan.name}" focused on "{plan.focus}":
{plan.description}

Rank the following longterm tasks by urgency and relevance to the plan's focus.
Return ONLY a JSON array of task IDs in priority order (highest urgency first).

Tasks:
{for each item: "- ID: {id}, Description: {description}"}
```

4. Call `ai.Generator` (same OpenAI integration used by insights)
5. Parse response as `[]uuid.UUID`
6. Reorder longterm items by returned ranking
7. Pass ranked list to `ScheduleMonth()` — the deterministic scheduler places them in rank order

### Fallback

If LLM call fails, fall back to ordering by `sequence` (original user ordering).

---

## 8. API Request/Response Shapes

### GET `/api/plans/:id/calendar?month=2026-03`

**Response:**

```json
{
  "statusCode": 200,
  "message": "Calendar entries retrieved successfully",
  "result": {
    "planId": "a1b2c3d4-...",
    "month": "2026-03",
    "days": [
      {
        "date": "2026-03-01",
        "entries": [
          {
            "id": "e1f2a3b4-...",
            "entryType": "daily",
            "position": 1,
            "pinned": false,
            "checklistItemId": "c1d2e3f4-...",
            "description": "Write unit tests for auth middleware",
            "done": false
          },
          {
            "id": "f5g6h7i8-...",
            "entryType": "longterm",
            "position": 2,
            "pinned": true,
            "checklistItemId": "d4e5f6g7-...",
            "description": "Design database schema for notifications",
            "done": false
          },
          {
            "id": "g9h0i1j2-...",
            "entryType": "recommendation",
            "position": 3,
            "pinned": false,
            "checklistItemId": null,
            "recTitle": "Go Concurrency Patterns",
            "recUrl": "https://example.com/video/123",
            "recDescription": "Deep dive into goroutines and channels"
          }
        ]
      },
      {
        "date": "2026-03-02",
        "entries": []
      }
    ]
  }
}
```

### PATCH `/api/plans/:id/calendar/:entry_id/move`

**Request:**

```json
{
  "targetDate": "2026-03-15",
  "targetPosition": 3
}
```

**Response:**

```json
{
  "statusCode": 200,
  "message": "Entry moved successfully",
  "result": {
    "id": "e1f2a3b4-...",
    "scheduledDate": "2026-03-15",
    "position": 3,
    "pinned": true
  }
}
```

### PATCH `/api/plans/:id/calendar/:entry_id/reorder`

**Request:**

```json
{
  "targetPosition": 5
}
```

**Response:**

```json
{
  "statusCode": 200,
  "message": "Entry reordered successfully",
  "result": {
    "id": "e1f2a3b4-...",
    "scheduledDate": "2026-03-10",
    "position": 5,
    "pinned": true
  }
}
```
