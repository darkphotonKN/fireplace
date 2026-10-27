---
id: I-02JBZ-7
status: done
implements: FS-02JBZ
blocked_by: [I-02JBZ-5, I-02JBZ-6]
labels: [feature]
title: "FS-02JBZ slice 7: the revision ladder for learning plans (+1/+3/+7, untick undoes, per-item switch)"
---
Implements FS-02JBZ §Requirements (R7 revisions, R10, R17 revisions, R37 revisions, R41–R49), §API surface (setItemRevisions, DayPlanEntry.revision, DayPlanEntry.revisionsEnabled)

## What to Build

Timing by rule, never by the model (ADR-0019 clause 5, D6). Server and gateway.

**insights-service.**

- **Learning only (R41).** Plan type from `GetPlan` (I-02JBZ-4). Project plans never schedule or
  offer revisions.
- **Start (R42).** An item's first day tick **ever** (this user, this plan) schedules revision 1
  due on the tick's date + 1, unless switched off.
- **Climb (R43, S6).** Ticking an entry stamped `revision = N` completes revision N and schedules
  N+1: + 3 days after that tick for N = 1, + 7 days for N = 2; N = 3 ends the ladder. A tick of an
  item with no revision due schedules nothing. Dates are the day plan's local dates.
- **Due (R44, S7).** Revision N is due for a date when its due date ≤ that date and it is
  incomplete; it stays due until ticked. Due revisions of archived or deleted items are skipped;
  **done items still revise**.
- **Server stamps (R10, R45).** Every `existing_item` entry whose item has revision N due carries
  `revision = N`, set by the server after validation (AI and rules plans). Due revisions are also
  a valid candidate even when the item is done (extend R8's candidate check).
- **Prompt (R7).** Due revisions are injected as top-priority candidates labelled "revision N";
  due revisions join the hash input (R17).
- **Rules (R37).** Feed due revisions into I-02JBZ-5's ranker (they rank first, "Revisions"
  group).
- **Untick undoes (R46, D5).** Untick of the ladder-starting tick removes revision 1 (and the
  item has no first tick again); untick of a revision-N tick reopens N and removes N+1 if that
  tick scheduled it. Hook into I-02JBZ-6's untick.
- **Switch (R47, R48).** Per (user, plan, item), default on. Off cancels the open rung and
  schedules none; on schedules nothing itself, and the next tick schedules rung
  (completed + 1), max 3. Item not in plan / plan not accessible → `NotFound`. Accepted on project
  plans, no effect.
- **Read model (R49).** `revisionsEnabled` on `existing_item` entries of learning plans only.
- **RPC** `SetItemRevisions`.

**Gateway.** Typed Huma `setItemRevisions` (`PUT /api/insights/day-plan/items/{itemId}/revisions`,
body `{planId, enabled}`, `204`). Regenerate `openapi.yaml` and the client; gates green.

## Acceptance Criteria

- [ ] Learning plan: first tick → revision 1 due date + 1; ticking revision 1 → revision 2 due that
      date + 3; ticking revision 2 → revision 3 due that date + 7; ticking revision 3 → nothing.
- [ ] Project plan: nothing scheduled, no `revision`, no `revisionsEnabled`.
- [ ] A due revision stays due until ticked; offered for a done item; not for an archived or
      deleted item.
- [ ] Unticking the first tick removes revision 1; unticking a revision tick reopens it and
      removes the next rung.
- [ ] Switch off cancels the open rung; switch on lets the next tick schedule the next rung, never
      past 3.
- [ ] Entries carry server-set `revision = N`; the model schema has no revision field; the prompt
      lists due revisions as "revision N".
- [ ] Rules plan puts due revisions first in a "Revisions" group.
- [ ] `setItemRevisions` for an item outside the plan → `404 · NOT_FOUND`.
- [ ] Regenerated contract; gates green.
- [ ] Tests pass.

## Blocked By

- I-02JBZ-6 — ticks and untick are what the ladder reacts to.
- I-02JBZ-5 — the rules ranker that due revisions feed.

## Expected write set

- `common/api/proto/insights/insights.proto` + regenerated stubs **(hotspot)**
- `services/insights-service/internal/dayplan/` (revisions repository, ladder, stamping, prompt, rules input) (+ tests)
- `services/insights-service/migrations/00000N_create_revisions.{up,down}.sql` (new: revisions + per-item switch)
- `services/insights-service/config/services.go` **(hotspot)**
- `services/api-gateway/internal/insights/typed_insights.go` (+ test), `services/api-gateway/internal/gateway/insights/client.go`
- `services/api-gateway/openapi.yaml` + `client/src/api/generated/schema.d.ts` **(hotspot, regenerated)**

## Spec Reference

FS-02JBZ §Decisions D5, D6; §Requirements R7, R8 (revision candidates), R10, R17, R37, R41–R49;
§Spec defaults S6, S7; §API surface (setItemRevisions, DayPlanEntry); §User Stories 16, 19–24,
36; §Edge States ("Tick then untick the same day, first tick", "Untick a revision tick",
"Revision slips several days", "Item done, revision due", "Revisions switched off with one
open", "Plan type changed"); ADR-0019 clause 5.

## TDD Approach

- RED: a ladder table test on a pure function (first tick, rung ticks, slipped rung, untick of
  each, switch off/on) → scheduled/removed rungs and due dates.
- GREEN: the ladder.
- Then: repository round-trip; stamping on AI and rules entries; prompt golden test with due
  revisions; project-plan no-op; handler and regen.
