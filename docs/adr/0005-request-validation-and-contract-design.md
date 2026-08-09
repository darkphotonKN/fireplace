# ADR-0005 — Request validation: shape at the boundary, domain downstream; strictness follows the deployment model

Status: accepted
Date: 2026-08-07
Scope: root — governs every serialized HTTP endpoint (plane 1) and every proto message (plane 2)
Realized by: FS-0002 (first application; the rules below are all covered by tests in `config/`)
Related: ADR-0002 (contract planes), ADR-0003 (transport types), ADR-0004 (error representation), ADR-0001 (identity)

## Context

Serializing the first endpoint forced three questions that ADR-0002 left open, and getting
them wrong is expensive in opposite directions.

**Where does validation live?** The gateway is a proxy: `auth-service` owns the domain rules
(its only profile rule is "name must not be empty"). If the gateway re-validates, the rule
exists in two places and drifts. If it validates nothing, malformed input costs a network hop
before failing.

**How strict should a request body be?** huma defaults to `additionalProperties: false`, so an
undeclared member is a `422`. The alternative — ignoring it — was tried and rejected during
FS-0002: on a `PATCH`, silently ignoring a member means `{"biio": "my new bio"}` returns
**200 OK and changes nothing**. The user believes they saved something they did not. That
failure is invisible; a `422` names itself.

The counter-argument is forward compatibility: strict request bodies couple client and server
releases, because a newer client sending a field an older server doesn't know is rejected
outright. That argument is decisive for consumers that deploy independently, and near-worthless
for a client that ships in the same release and consumes generated types — it cannot emit an
undeclared field by accident.

**What may a client send at all?** `id`, `createdAt` and `updatedAt` are real fields of a
profile and appear in every response, but none of them are writable. A client echoing back a
resource it just fetched is sending fields the server itself produced.

## Decision

### 1. Two validation layers, two status codes, no overlap

| Layer | Question | Where | Status | Body |
|---|---|---|---|---|
| **Shape** | is this even a well-formed request for this operation? | the boundary — huma, from the type | **422** `VALIDATION_FAILED` | problem+json |
| **Domain** | is this request *allowed* given the rules of the domain? | the **owning service**, never the gateway | **400** + a specific domain code | problem+json |

The gateway never restates a domain rule. `name: ""` travels to `auth-service` and returns
`400 PROFILE_NAME_EMPTY`; the gateway only maps the gRPC status to HTTP and attaches the code.

### 2. Strictness follows the deployment model — and therefore differs per plane

**The rule: reject unknown input when the consumer ships with you; tolerate it when it does not.**

- **Plane 1 (HTTP, first-party web client): STRICT.** Request bodies keep
  `additionalProperties: false`. An undeclared member is a `422` and never reaches the
  downstream service. Safe because the client ships in the same release and consumes the
  generated TypeScript client, so the only way to send an undeclared field is deliberately.
  Typos surface at `tsc` for the real client and at `422` for everyone else.
- **Plane 2 (gRPC, service-to-service): TOLERANT — and this is not an inconsistency.**
  Protobuf ignores unknown wire fields by design, precisely so services can be rolled out
  independently. We do not fight that. Plane 2's equivalent protection is not runtime
  rejection but `buf breaking` in CI: the schema cannot change incompatibly in the first place.

Same goal — no silent drift — enforced where each plane can actually enforce it.

**Revisit trigger for plane 1:** the moment a consumer appears that does *not* deploy with the
server (a mobile app, a third party, a public API), strictness on request bodies becomes a
release-coupling liability and this decision must be re-taken.

### 3. Request types declare exactly what a client may send

- **Read-only fields never appear in a request type.** `id`, `createdAt`, `updatedAt` are
  response-only. A request type is not a resource; it is the set of writable fields.
- **Identity is never a body field.** It comes from the JWT `sub` claim (plane 1) or verified
  metadata (plane 2, ADR-0001). Accepting a client-supplied id on "update *my* profile" is the
  trust-the-body-field hole ADR-0001 exists to close — so `id` is absent by design, not omission.
- **Optional means `omitempty`.** huma marks a field **required** unless its json tag carries
  `omitempty`. A partial-update type is therefore `*T` + `json:"…,omitempty"` on every field.
  Without it the generated schema requires all of them and an empty body `{}` — a legitimate
  no-op — is rejected.
- **Responses may stay closed** (`additionalProperties: false`). A server never validates its
  own response, so this only documents the exact published shape and costs nothing.

## Consequences

**Accepted / positive:**

- A mistyped field fails loudly instead of returning 200 and doing nothing — the single most
  valuable property of this decision.
- One place per rule: shape at the boundary, domain in the owner. Neither can drift from the
  other because neither duplicates the other.
- `422` vs `400` is a meaningful signal to a client: 422 means *your request is malformed*,
  400 means *your request is fine but the domain refused it*. The domain code says which rule.
- Read-only fields being absent from request types makes a whole class of mass-assignment bug
  unrepresentable rather than defended against.
- Plane 2 keeps protobuf's rolling-upgrade property, which is the reason to use protobuf.

**Costs / follow-ups:**

- **Plane 1 couples client and server releases.** Adding a field to a request type means older
  clients that don't send it are fine, but newer clients that do send it break against older
  servers. Acceptable only while the client ships with the server — hence the revisit trigger.
- **The round-trip PATCH pattern is forbidden** on plane 1: a client must send only writable
  fields, not the whole resource it just fetched. The generated client makes this natural, but
  it is a real constraint on hand-written consumers and must be documented for them.
- Two different strictness policies across the two planes need explaining to anyone who sees
  only one of them — hence this ADR rather than a code comment.
- `FS-0002 §Edge States` currently claims unknown fields are "ignored, not an error." That row
  predates this decision and is **wrong**; it needs a `write-a-spec` amendment.

**Alternatives rejected:**

- **Ignore unknown members (tolerant reader) on plane 1.** Preserves forward compatibility and
  supports round-trip PATCH — but converts client bugs into silent no-ops. Correct for a public
  API with uncontrolled consumers; wrong for a first-party client that ships in lockstep.
- **Validate domain rules at the gateway too** (fail fast, save a hop). Duplicates every rule
  across two services and guarantees drift. The hop is cheap; the drift is not.
- **Move domain failures to 422 as well**, for one uniform "invalid" status. Collapses a
  genuinely useful distinction — malformed vs refused — and would change a live endpoint's
  status code for no gain.
