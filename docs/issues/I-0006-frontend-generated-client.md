---
id: I-0006
status: open
implements: FS-0002
blocked_by: [I-0004, I-0005]
labels: [enhancement]
title: FS-0002 slice 5: frontend cutover to the generated TypeScript client
migrated_from: github#51
---
Implements FS-0002 §Requirements, §API surface

## What to Build

The frontend cutover — where the deliberate break actually lands in the UI.

- Generate the TypeScript client (`openapi-typescript` + `openapi-fetch`) into
  `client/src/api/generated`. This directory does not exist yet; it is created here.
- Migrate the profile **read** and **update** calls off hand-written fetch
  (`client/src/services/api.ts`, and any profile calls in `client/src/services/`) onto the
  generated client.
- Handle the two shape changes for these two paths **only**:
  - success is a **bare resource** — no `result` unwrapping;
  - errors are **problem+json** — branch on `code`, not on a message string.
- **Legacy endpoints keep the envelope and their existing fetch calls.** Do not migrate them
  (ADR-0002 grandfather clause).

## Why this is one slice with both operations

The grandfather clause requires the FE call to migrate in the *same feature* that serializes
its endpoint. Both profile operations serialize in this feature, so both cut over together.

## Acceptance Criteria

- [ ] `client/src/api/generated` exists and is produced by a repeatable command
      (`make client` from I-0001), not hand-written.
- [ ] Profile read and update both go through the generated client.
- [ ] **No hand-written fetch to `/api/users/profile` remains** — after this slice, such a
      call is a HIGH code-review finding.
- [ ] Error handling branches on the `code` extension, not on `detail` or a status alone.
- [ ] A `PROFILE_NAME_EMPTY` response surfaces a field-appropriate message to the user
      **without relying on `errors[]`**, which is empty for downstream failures.
- [ ] `tsc` passes; a deliberate shape mismatch against the generated types fails the build
      (verify once, then revert).
- [ ] Calls to at least one legacy endpoint (e.g. plans) are untouched and still parse the
      `{statusCode, message, result}` envelope correctly.

## Blocked By

I-0004, I-0005 — both operations must exist in the committed spec before the client can be
generated from it. Client generation additionally needs `make client` from I-0001.

## Spec Reference

FS-0002 §Requirements 20 · §API surface (both rows — the generated client must match the
table exactly). Constraints: ADR-0002 §4 (generated-client-only rule) and §7 (grandfather
clause), ADR-0003 (bare resources), ADR-0004 (`code` is the switch key).

## Note for the reviewer

This is the first place the two success shapes coexist in the frontend — bare for profile,
enveloped for everything else. That is by design and has a known end state (the last
endpoint to serialize). It is not an inconsistency to "clean up" opportunistically.
