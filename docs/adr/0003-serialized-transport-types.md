# ADR-0003 — Serialized responses are bare resources backed by per-operation transport types

Status: accepted
Date: 2026-08-05
Scope: root — governs every serialized HTTP endpoint at the api-gateway
Realized by: FS-0002 (first application; the transport types land in its slice ⓪)

## Context

Recorded without adversarial review — both halves of this decision were locked in a
`scope-it` session and marked **(not challenged)** by user choice. They concern a public API
contract, which ADR-0002 itself classes as expensive to reverse, and they set the precedent
every subsequent serialize-on-touch slice inherits. The marker is deliberate.

Two forces surfaced while scoping FS-0002, the first endpoint to be serialized under
ADR-0002's adoption policy.

**The envelope.** Every gateway success response is wrapped in
`{statusCode, message, result}`. This was kept deliberately at the gRPC cutover so existing
frontend clients saw no change (`authgw/handler.go` says so explicitly). Serializing an
endpoint bakes that wrapper into the *typed schema*, where it stops being an implementation
habit and becomes published contract — inherited by every endpoint serialized after it.
It also sits badly beside ADR-0002's commitment to problem+json on errors: success wrapped
in a bespoke envelope, failure in a standard media type, is an asymmetry with no rationale
behind it other than history.

**The storage model.** `models.User` carries `Password string` with
`json:"password,omitempty"`. It is never populated by `userFromProto`, so it never ships
today — but a schema generated from that struct **publishes a `password` field**.
Serialization converts an invisible field into a documented part of the contract. The
general form of the problem: a storage struct's shape is governed by storage concerns
(columns, migrations, ORM tags), while a wire type's shape is governed by contract concerns
(what clients may depend on). Coupling them makes every future `ALTER TABLE` a silent
contract change, opt-out rather than opt-in.

Both forces point the same way: **the contract's shape must be chosen, not inherited.**

## Decision

1. **Serialized success responses are bare resources.** The typed schema is the resource
   itself, with no wrapper. Metadata that genuinely belongs in a response body (pagination
   cursors, etc.) is modelled explicitly per operation, not smuggled through a generic
   envelope.

2. **Transport types are declared per operation.** Each serialized operation names explicit
   request and response types owned by the contract layer (e.g. `ProfileResponse`,
   `UpdateProfileRequest`), living beside the handler that serves them.

3. **Storage models are never serialized.** `models.*` and other persistence entities must
   not appear in a generated schema — not as a response type, and not nested inside one.
   A transport type that embeds a storage struct reintroduces exactly the coupling this
   rule exists to prevent.

4. **Envelope removal is a deliberate, grandfathered break.** The frontend cuts over to the
   generated client for an endpoint in the same feature that serializes it (ADR-0002's
   grandfather clause). Legacy, unserialized endpoints keep the envelope until touched. The
   transition therefore has two live success shapes, by design and with a known end state.

## Consequences

**Accepted / positive:**

- The published contract is a designed artifact. A new database column does not enter the
  API unless someone puts it in a transport type.
- The `password`-class leak becomes impossible by construction rather than by the accident
  of `omitempty` and a mapping function that happens not to populate it.
- Success and error representations are symmetric — both are standard, neither is bespoke.
- Storage refactors are free: renaming a column or restructuring an entity cannot break a
  client, because no client ever saw the entity.
- Generated TS client consumers get precise types per operation instead of one permissive
  union with everything optional.

**Costs / follow-ups:**

- **Boilerplate:** one transport type plus a mapping function per operation. This is the
  price of the decoupling and should not be optimized away by sharing types (see rejected
  alternatives).
- **Two success shapes during the transition** — the FE handles enveloped legacy endpoints
  and bare serialized ones simultaneously until the last endpoint is touched.
- **A new review obligation:** a transport type that *nests* a storage struct passes
  compilation and silently violates rule 3. This belongs on `code-review`'s Standards axis,
  not left to vigilance.
- FS-0002's `ProfileResponse` publishes exactly `id, email, name, displayName, bio,
  createdAt, updatedAt` — it is the worked example; later operations should follow its shape.

**Alternatives rejected:**

- **Keep the envelope in the typed schema.** Zero FE change on the success path, but it
  makes a non-idiomatic wrapper permanent, published contract, inherited by every endpoint
  serialized afterwards, and preserves the success/error asymmetry.
- **Reuse storage models as wire types.** No new types to write, but the DB entity becomes
  the contract and future columns enter the API by default rather than by choice — the
  `password` finding is the concrete instance of this failure.
- **A shared DTO package reused across operations.** Fewer types than per-operation
  declaration, but a type shared by three operations has three reasons to change; the first
  operation needing a field it doesn't want forces either a permissive optional field or a
  fork. Per-operation declaration is the point of the rule, not an accident of it.
