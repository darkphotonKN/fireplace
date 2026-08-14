# FS-0005: Per-user authorization on plan-scoped resources

> Status: draft · SPECIFICATION.md: services/api-gateway/SPECIFICATION.md "## Authorization" → this FS · Related ADRs: docs/adr/0001-grpc-identity-via-metadata-rs256.md (the identity seam this builds on), docs/adr/0005-request-validation-and-contract-design.md (where authorization sits relative to validation)

## Scoping notes (raw)

### Status: notes FIXED; plans still open

The owner confirmed this is **unbuilt work that was always intended**, not an oversight
discovered late — and asked for it to be built rather than only recorded.

**The notes half is now done** (commit below): every notes handler asserts plan ownership, and
a two-account probe confirms a stranger gets 403 on list/create/get/delete while the owner is
unaffected. **The plans half is still open** — see "What remains".

The evidence below is recorded so it is not re-derived by whoever picks this up, and so the
contract stops implying a guarantee that does not exist.

Nothing about it is urgent in the current deployment: fireplace has effectively one real user,
so the exposure is latent rather than live. The severity is real; the exposure today is not.

### What is actually true right now (measured, 2026-08-15)

Probed against a running gateway with two real accounts — A owning a plan, B a stranger to it.
Not read off the code: an earlier reading of `AssertPlanOwnership` produced the *wrong*
conclusion, which is why this section reports observations.

| Action by non-owner B | Observed |
|---|---|
| `GET /api/plans/{A's plan}` | **200 — full plan, including A's userId** |
| `PATCH /api/plans/{A's plan}` | 200, but the change silently does nothing |
| `DELETE /api/plans/{A's plan}` | 404 — correctly refused |
| `GET /api/plans/{A's plan}/notes` | **200 — A's note content** |
| `POST /api/plans/{A's plan}/notes` | **201 — B's note written into A's plan** |

So: **plans leak on read; notes leak on read and write.** Delete is already scoped correctly,
which shows the intent exists in plan-service and is applied unevenly.

### What remains

`GET /api/plans/{id}` still returns another user's plan. The mechanism to fix it exists and is
proven to work — `AssertPlanOwnership` rejected a non-owner in the notes probe — so it is one
call in the gateway's typed handler. It was NOT done here for one reason: **shared plans**.
`GET /api/plans/shared` exists, so "owner" is not the only relation that should grant read
access, and asserting ownership on a plain read would break sharing. That question has to be
answered first; see Open questions.

### Root cause, per surface

- **Notes (gateway-owned).** No notes handler reads the caller's identity at all — not one
  calls `auth.GetUserID` or `auth.UserIDFromCtx` — and every repository query is scoped by
  `plan_id` alone. There is no user anywhere in the code path to check against. This is the
  gateway's own domain, so the fix is entirely local.
- **Plans / checklists (the GATEWAY, not plan-service).** This was my first, wrong reading.
  plan-service does not enforce ownership on direct reads **by design** — it treats the gateway
  as a trusted caller and exposes `AssertPlanOwnership` for the gateway to call where ownership
  matters. `plangw.Adapter`'s own comment states this. So the missing check is the gateway's,
  the same as notes, and `AssertPlanOwnership` is proven to work: it is what returns 403 to a
  stranger now that notes call it.

### Decisions taken

1. **Authorization is not validation.** It does not belong at the huma boundary as a shape
   rule (ADR-0005). Ownership is a domain question answered by whoever owns the data — the
   gateway for notes, plan-service for plans and checklists.
2. **403 is already declared** in the `Errors` list of every plan and checklist operation, so
   enforcing ownership later is **not a contract change**. That was deliberate.
3. **A silent no-op is its own bug.** `PATCH` by a non-owner returning 200 while changing
   nothing tells the caller they succeeded. Whatever this feature decides, that stops.

### Open questions

- **403 or 404 for a non-owner?** 403 says "this exists and is not yours", which leaks
  existence; 404 says nothing. Today the surface does both — DELETE 404s while the declared
  contract says 403. One answer, applied everywhere.
- **Are shared plans in scope?** `GET /api/plans/shared` exists, so "owner" is not the only
  relation that grants access. The rule is probably *owner or shared-with*, and that needs
  settling before any check is written, or sharing breaks.
- **Does notes ownership derive from the plan?** Most likely — a note belongs to whoever may
  see its plan — which makes it a plan-service ownership question the gateway asks, not a
  second rule.

### Out of scope for this note

The fix itself. This is scoping residue captured while FS-0004 was in flight; promote it with
`write-a-spec` before implementing.
