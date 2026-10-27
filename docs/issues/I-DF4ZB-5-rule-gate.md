---
id: I-DF4ZB-5
status: done
implements: FS-DF4ZB
blocked_by: [I-DF4ZB-1, I-DF4ZB-4]
labels: [feature]
title: "FS-DF4ZB slice 5: the rule gate, plan-state hash carry-forward and shared-only retrieval"
---
Implements FS-DF4ZB §Requirements (R18 second and third clauses, R19)

## What to Build

Skip the expensive work by rule when nothing changed (L6). Runs before any embedding, retrieval
or model call.

- **Plan-state hash (R19).** A digest over the R1 plan tree (each item's id, parent id,
  description, done, due date), ordered by id, so two trees the model would see identically hash
  identically. Stored with every brief.
- **Carry-forward (R18, second clause).** When there is no brief for today's key and the hash
  equals the hash on the user's most recent brief for this plan, serve that brief's moves copied
  forward to today's key with no generation. `generatedAt` keeps the original draw time.
- **Shared-only retrieval (R18, third clause).** If the user has no personal-memory rows
  (I-DF4ZB-1's repository read), the retrieval seam is asked for shared rows only. With the stub
  retriever this is observable as the scope passed to the seam.
- **The dismissal bypass is not here.** "…and none of that brief's moves has been dismissed" needs
  feedback, which lands in I-DF4ZB-10; that slice extends this gate. Leave a clear extension
  point.

Built against the R29 seam/stub. Not blocked by H2.

## Acceptance Criteria

- [ ] Two plan trees that differ only in order of rows hash identically; changing any one of id,
      parent id, description, done or due changes the hash.
- [ ] A read on a new day with an unchanged plan makes no model call and returns the previous
      brief's moves under today's key, with the original `generatedAt`.
- [ ] A read on a new day after the plan changed generates.
- [ ] A user with no personal rows triggers no personal retrieval (the seam is called with shared
      scope only).
- [ ] Tests pass.

## Blocked By

- I-DF4ZB-4 — the brief flow and storage the gate sits in front of.
- I-DF4ZB-1 — the "does this user have personal rows" read.

## Expected write set

- `services/insights-service/internal/guided/` (gate, hash, brief service and repository)
- `services/insights-service/migrations/00000N_*` (only if I-DF4ZB-4 did not already store the hash)
- `services/insights-service/config/services.go` **(hotspot)**, if the gate needs the rag
  repository injected

## Spec Reference

FS-DF4ZB §Requirements R18, R19; §Decisions L6; §Spec defaults S3; §Edge States ("Plan edited
after the brief was drawn", "New user, no personal memory"); §Acceptance Criteria ("A read on a
new day with an unchanged plan…", "A user with no personal rows…").

## TDD Approach

- RED: a hash table test (reordered rows equal; each field change differs).
- GREEN: the digest.
- Then: a brief-service test: yesterday's brief stored with hash H, today's tree hashes to H →
  zero generator calls and moves carried forward; then a changed tree → one call.
