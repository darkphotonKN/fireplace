---
id: I-0005
status: done
implements: FS-0002
blocked_by: [I-0004]
labels: [enhancement]
title: FS-0002 slice 4: PATCH /api/users/profile serialized (UpdateProfileRequest, null semantics)
migrated_from: github#50
---
Implements FS-0002 §Requirements, §Edge States, §API surface

## What to Build

`PATCH /api/users/profile` as a typed Huma handler, reusing `ProfileResponse` from I-0004.

- **`UpdateProfileRequest`** — `name`, `displayName`, `bio`, all optional.
- **Typed handler** returning the updated profile as a bare `ProfileResponse`.
- **No edge validation constraints in the schema.** The schema pins shape only; validation
  stays in `auth-service`.

## Semantics to preserve exactly (this is documentation, not redesign)

The repository builds a dynamic `SET` clause, so a nil field is omitted. That produces:

| Request | Behavior |
|---|---|
| `{"bio": "hi"}` | sets `bio` |
| `{"bio": ""}` | sets `bio` to the empty string |
| `{"bio": null}` | **ignored — unchanged** (null is indistinguishable from absent) |
| `{}` | `200`, profile unchanged |
| `{"name": ""}` | `400`, `code: PROFILE_NAME_EMPTY` |
| unknown fields | ignored, not an error |

**`name: ""` must return 400, NOT 422.** Declining edge validation is deliberate — declaring
`minLength:1` would make Huma reject at the edge with 422, moving the status code on a live
endpoint. Validation stays downstream so the status stays 400.

**NULL-clearing is explicitly out of scope** — it would require a nullable wrapper at the
gateway plus a proto change and an auth-service repository change, moving both contract
planes during the validation run.

## Acceptance Criteria

- [ ] `PATCH` returns `200` with the updated bare `ProfileResponse`.
- [ ] Every row of the semantics table above is covered by a test.
- [ ] `{"name": ""}` returns `400` with `code: PROFILE_NAME_EMPTY` — asserted to be 400, not 422.
- [ ] `errors[]` is present and empty for the downstream validation failure (documented
      limitation, not a bug — see I-0003).
- [ ] The operation appears in the generated `openapi.yaml` with no declared length/format
      constraints on the three fields.
- [ ] `password` still appears nowhere in the generated document.

## Gate-dependent (blocked on I-0001)

- [ ] Regenerating the spec produces no diff against the committed `openapi.yaml`.
- [ ] Spectral passes on the operation.
- [ ] `oasdiff` reports the envelope removal as breaking and it is allowlisted with a reason.

## Blocked By

I-0004 (shares `ProfileResponse`). Gate-dependent acceptance criteria additionally require I-0001.

## Spec Reference

FS-0002 §Requirements 7–12, 16 · §Edge States · §API surface (`updateProfile` row).

## TDD Approach

- RED: table test over the six semantic rows above, asserting body and status for each.
- GREEN: `UpdateProfileRequest` + typed handler + mapping.
- RED: assert the generated schema declares no constraints on `name`/`displayName`/`bio`.
