# FS-0004 R14 — before/after probe for envelope removal

**Run:** 2026-08-14 · baseline `b56ad32` on :6061 (git worktree) vs serialized HEAD on :6060 ·
same Consul, same auth-service (7101), same plan-service (7103), same databases.

## Why this file exists

Envelope removal is a **deliberate break the breaking-change ratchet structurally cannot see**.
These operations were not in the previous OpenAPI document, so there is no diff for oasdiff to
compare against. `make gates` passing says nothing about it either way.

FS-0004 R14 therefore requires a live before/after probe as the *only* end-to-end evidence, and
this is that evidence. It is recorded rather than described, because "I ran it and it looked
fine" is not a check anyone can re-examine.

## Method

The legacy code is deleted by this feature, so the "before" cannot come from the running tree.
It was stood up from git at `b56ad32` — the last commit before the users group was serialized —
in a worktree, pointed at the same infrastructure, on port 6061. Both gateways were live
simultaneously and answered the same requests with the same bearer token.

One probe account (`fs0004-probe@example.test`) was created for this, because an authenticated
before/after pair cannot be obtained otherwise.

## Results

### `POST /api/users/signin`

```
BEFORE top-level keys: ['message', 'result', 'statusCode']
AFTER  top-level keys: ['accessExpiresIn','accessToken','refreshExpiresIn','refreshToken','userInfo']
```

Envelope removed. Payload compared field by field:

| field | result |
|---|---|
| `accessExpiresIn`, `refreshExpiresIn` | identical |
| `accessToken`, `refreshToken` | differ — freshly minted per sign-in, not a shape change |
| `userInfo` | identical **apart from the declared date rename** |

```
userInfo BEFORE: ['bio','created_at','displayName','email','id','name','updated_at']
userInfo AFTER:  ['bio','createdAt','displayName','email','id','name','updatedAt']
```

### `GET /api/users`

```
BEFORE: {statusCode, message, result: [...]}     AFTER: bare array
7 users before, 7 after
one record: identical apart from the declared date rename
```

### `GET /api/plans`

First run returned an empty list, which would have made this check **vacuous** — an empty array
compares equal to an empty array and proves nothing about record shape. A plan was created via
the baseline and the probe re-run:

```
BEFORE: {statusCode, result: [...]}              AFTER: bare array
1 plan before, 1 after
before keys: created_at, dailyReset, description, focus, id, name, planType, updated_at, userId
after  keys: createdAt,  dailyReset, description, focus, id, name, planType, updatedAt,  userId
verdict: identical apart from the declared date rename
```

### Incidental confirmations

- **`password` appears nowhere** in any serialized users or plans body. `models.User` carried
  `json:"password,omitempty"` and was the legacy response type for all three users reads; the
  probe confirms the hazard is gone in practice, not only by construction.
- **`DELETE /api/plans/{id}` answers 204 with a zero-byte body** — verified while removing the
  probe plan, and the subsequent list returned `[]` rather than `null`.
- Both surfaces returned `[]` and never `null` for empty lists.

## Verdict

The two declared breaks are the only differences, and both are the ones this feature set out to
make: **the envelope is gone**, and **date keys are camelCase**. Nothing else about any payload
changed.

## Cleanup

The baseline gateway was stopped, the worktree removed, and the probe plan deleted. The probe
**user account was left in place** — the API exposes no user-deletion endpoint, so removing it
would mean touching the auth database directly, which is not this feature's business. It is
inert: `fs0004-probe@example.test` in the dev auth DB.

---

# I-0017 — checklists

**Run:** 2026-08-14 · baseline `6c92a23` on :6061 vs serialized HEAD on :6060, same infrastructure.

## Envelope removal

```
GET /api/plans/{id}/checklists?scope=daily
BEFORE: {statusCode, result: [...]}          AFTER: bare array
1 item before, 1 after
before keys: archived, created_at, description, done, id, planId, scope, sequence, type, updated_at
after  keys: archived, createdAt,  description, done, id, planId, scope, sequence, type, updatedAt
verdict: identical apart from the declared date rename
```

## The three-state `parentId`, end to end

This is the part of I-0017 that most needed live proof. The schema hook changes how the field
is *expressed* in the contract, and JSON Schema cannot represent "omitted" at all — so a
document that looks right is not evidence that all three states still work.

| Sent | Expected | Observed |
|---|---|---|
| `{"parentId":"<uuid>"}` | re-parent | `parentId` = that uuid ✓ |
| `{"description":"…"}` (parentId omitted) | parent unchanged | `parentId` still the uuid, description updated ✓ |
| `{"parentId":null}` | clear to top level | `parentId` absent ✓ |

Omit and null are distinguishable through the published schema, which is what the
`huma.SchemaProvider` hook exists to preserve.

## The one behaviour change, confirmed live

```
GET /api/plans/{id}/checklists?scope=bogus
-> 422  code: VALIDATION_FAILED
```

The legacy handler forwarded any non-empty scope to plan-service. The declared enum now
rejects it at the boundary. Correct per ADR-0005 — scope is shape, shape is validated at the
edge — but a change, not a transcription.

**Asymmetry worth a decision:** the same value inside a request BODY is still forwarded
downstream, because the request types keep their (huma-ignored) `enums:` tags. Making the two
consistent means adding boundary validation to bodies, which is a further behaviour change
rather than a cleanup.

## Cleanup

Probe plan deleted via the serialized `DELETE` (204). Baseline stopped, worktree removed.
