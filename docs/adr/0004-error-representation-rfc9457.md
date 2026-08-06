# ADR-0004 — Error representation: RFC 9457 problem+json with a domain `code` and field-level `errors[]`

Status: accepted
Date: 2026-08-05
Scope: root — governs every serialized HTTP endpoint at the api-gateway
Amends: **ADR-0002 § error clause** (which specifies "RFC-7807 problems" and assumes the
stock error model). ADR-0002 stands in every other respect; only its error representation
is replaced here.
Realized by: FS-0002 (first application; the custom error model lands in its slice ⓪)

## Context

Recorded without adversarial review — locked in a `scope-it` session and marked
**(not challenged)** by user choice. Error representation is public contract and is
switched on by client code, so it is expensive to change later. The marker is deliberate.

ADR-0002 committed the gateway to "RFC-7807 problem responses." Two problems with that
clause surfaced when the first endpoint was actually scoped.

**The RFC is obsolete.** RFC 9457 (July 2023) obsoletes RFC 7807. The media type and core
member names are unchanged, so this is not a rewrite — but the citation should name the
standard that is actually current, and 9457 is explicit about extension members in a way
7807 was not.

**Status alone is too coarse to program against.** The gateway's `apierr.StatusFor` maps
everything to an HTTP status and a deliberately generic, client-safe message. A single 400
currently covers a malformed request body, an unparseable UUID, and an empty name. A
frontend that needs to react differently to those has nothing to switch on except the
`detail` string — which is prose, is not contract, and is explicitly allowed to change.
The result is either brittle string matching or an FE that cannot distinguish failures at
all. The spike (`docs/notes/contract-pioneer-log.md`, 2026-08-04) confirmed the stock
`huma.ErrorModel` carries `type`, `title`, `status`, `detail`, `instance`, `errors` — and no
domain identifier.

## Decision

1. **RFC 9457 problem+json is the error representation** for every serialized endpoint,
   served as `application/problem+json`.

2. **Two extension members are carried on every problem response:**
   - **`code`** — a domain error code in `SCREAMING_SNAKE`, stable enough that frontend code
     switches on it. This is the precise failure identifier.
   - **`errors[]`** — field-level detail: message, location (`body.items[3].tags`,
     `path.thing-id`), and the offending value.

3. **HTTP status stays the coarse routing signal; `code` is the precise one.** Status keeps
   its RFC-defined meaning and never encodes domain specifics. Two failures that are both
   "the request was invalid" share a 400 and differ by `code`.

4. **`code` is contract.** Removing or repurposing a code is a breaking change, reviewed
   like removing a response field. Adding a new code is non-breaking, so handlers may become
   more specific over time without a coordinated release.

## Consequences

**Accepted / positive:**

- The frontend switches on a stable token instead of string-matching prose, and gets a
  compile-time-visible set of failures through the generated client.
- Field-level `errors[]` is **free** — `huma.ErrorDetail` already models
  message/location/value and the stock `NewError` populates it.
- Handlers can grow more precise (new codes) without breaking existing clients.
- One error representation across every serialized endpoint; no per-domain error dialects.

**Costs / follow-ups:**

- **A custom error model is required.** `huma.ErrorModel` has no extension field for `code`,
  so the gateway must define its own type and override the `huma.NewError` var (a documented
  hook, global to the service). The custom type must implement `GetStatus()`, `Error()`,
  **and `ContentType()`** — the stock model's `ContentType` is what emits
  `application/problem+json`, and a custom model that omits it silently degrades to
  `application/json`, passing tests that only assert on status and body.
- **`apierr.StatusFor` must grow a domain-code dimension.** It returns `(int, string)` today
  and there is no domain code anywhere in the gateway. This is the gateway-wide error
  boundary, so it is a structural change, not a local one — and it is the seam every future
  serialized endpoint depends on.
- **The code vocabulary itself is undefined and blocking.** Who owns it, whether it is
  per-service or platform-wide, and whether it belongs in `common/` are open questions that
  must be answered before FS-0002's slice ⓪ can finish. When they are, the vocabulary is
  ubiquitous language and belongs in `CONTEXT.md` via `domain-model`.
- **`§API surface` tables gain domain codes per error row** — a format change to
  `docs/specs/README.md` (the spec-system authority), not to any single FS.
- **The spike's adapter finding is amended.** `problemFor` was a two-liner *because* it
  returned the stock model; under this ADR it becomes a custom type plus a `NewError`
  override. Still one seam, but the pioneer log's "two-liner" note needs this caveat.

**Alternatives rejected:**

- **Stock `ErrorModel` only (status + detail).** No custom type, no override — but it leaves
  the FE string-matching a prose field that is explicitly not contract. This is the status
  quo's failure, restated in a standard media type.
- **Use the RFC's own `type` URI as the switch key.** This is the standard-sanctioned
  identifier and needs no extension member at all — a real point in its favour. Rejected
  because a URI is awkward as a discriminant in TypeScript and invites accidental coupling
  to a hostname or path that may later move; a flat `SCREAMING_SNAKE` token is stable and
  cheap to switch on. `type` remains available and deliberately unused, so this is
  reversible if the URI approach proves better.
- **A bespoke non-standard error envelope.** Full freedom over shape, but discards the
  RFC's interop and tooling and re-opens exactly the question this ADR closes.
- **Encoding domain meaning in unusual HTTP statuses** (e.g. 422 vs 400 to distinguish
  failure kinds). Overloads a coarse, proxy-visible signal with domain vocabulary, and
  couples the domain's error taxonomy to HTTP's. FS-0002 explicitly declined a 400→422 move
  for this reason.
