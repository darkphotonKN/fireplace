# API documentation conventions

> Scope: the **api-gateway** service — the only HTTP surface in Fireplace. The
> other services (`auth-service`, `plan-service`, `calendar-service`) speak gRPC
> and are not documented with OpenAPI.

## Governing principle (non-negotiable)

- **Code is the single source of truth for validation.** Runtime validation lives
  in Go: `binding` struct tags at the HTTP edge, and **domain invariants in the
  downstream services**. Validation logic never moves into the spec.
- **The OpenAPI spec is a *generated description of shape*, never hand-maintained
  and never a validator.** It is generated FROM the Go edge types (code-first) with
  `swaggo/swag`, so it cannot drift from the code. `docs/swagger.{json,yaml}` and
  `docs/docs.go` are **build artifacts** — never edit them by hand.
- **Never encode conditional business rules in the schema** (no `if/then`, no
  `oneOf` for "field X required when Y"). Conditional rules are described in **prose**
  in the field's / endpoint's doc comment and enforced in Go. `enum`, `type`,
  `format` are fine — those are pure shape.

## Target version: OpenAPI / Swagger 2.0 (only)

Output is **Swagger 2.0** (the `swaggo/swag` default). We do **not** emit or convert
to 3.x. 2.0 is a deliberate choice: we use the spec as a **shape contract, not a
validator**, and 2.0's lack of conditional constructs (`if/then`, conditional
`oneOf`) structurally reinforces the principle that conditional rules live in Go,
not the schema. A `required_if`-style rule therefore appears in 2.0 as an
**optional** field whose `description` carries the rule — that is intended, not a
bug.

## Detected architecture & conventions (follow these — don't introduce new ones)

| Aspect | What this repo does |
|---|---|
| **Architecture** | api-gateway is an **inbound HTTP→gRPC adapter**, not a domain service. It owns no domain; the domain + conditional validation live in the downstream gRPC services. So edge DTOs do **shape/presence** validation only; domain rules are enforced downstream and *described in prose* here. |
| **Edge DTOs** | Already separated from domain. Request/response structs live in the handler packages (`internal/gateway/plan`, `internal/gateway/auth`, `internal/notes`, …). Handlers never serialize domain/protobuf types directly — they map via `model.go` + `adapter.go`. |
| **Naming** | Position-named: **`XxxReq` / `XxxResp`** (plan, notes). `auth` uses `XxxRequest`/`XxxResponse`. Follow the convention of the package you're editing. |
| **Router** | gin. Docs UI is **gin-swagger**, served at `/swagger/index.html`. |
| **Validation lib** | go-playground/validator **via gin `binding` tags** (`ShouldBindJSON`/`ShouldBindQuery`). No direct validator calls, no standalone `validate:` tags. `swag` reads `binding:"required"` for required-ness. |
| **Response shape** | Standard gin envelope: `{ "statusCode": <int>, "result": <payload> }` on success, `{ "statusCode": <int>, "message": <string> }` on error (`internal/apierr`). Documented via the doc-only `*Response` / `ErrorResponse` envelope structs in `model.go`. |

### Reference vertical

The fully-annotated reference is the **checklists** group in
`internal/gateway/plan/handler.go` + its DTOs in `internal/gateway/plan/model.go`.
The conditional-rule exemplar is **`PATCH /plans/{id}/checklists/{checklist_id}/dates`**:
`startDate`/`dueDate` are both optional in the schema, and the rule "`startDate` must
be ≤ `dueDate` when both are present" lives in the endpoint's prose `@Description`
and is enforced in `plan-service` — never as a schema constraint.

## Tooling

| Command | Does |
|---|---|
| `make gen` (≡ `go generate ./...`) | Regenerate the spec from annotations via `swaggo/swag`. |
| `make lint` | Spectral lint of `docs/swagger.json` (catches example-vs-schema drift). Needs Node/`npx`. |
| `make docs` | Serve the spec in a standalone local UI (Redoc on `:8089`). |
| running gateway | gin-swagger UI at `http://localhost:8080/swagger/index.html`. |

> **Spectral tradeoff:** `make lint` needs Node. For a pure-Go setup, drop Spectral
> and rely on `make gen` (swag's own parse/validation) succeeding — a weaker net
> that won't catch example-vs-schema mismatches as precisely.

## How to add a new endpoint (5 steps)

1. **Define edge DTOs** in the handler package's `model.go`, following the local
   `Req`/`Resp` naming. Put **shape** in struct tags only: `json`, `example`,
   `enums`, `format`, `swaggertype` (for custom types like `OptDate`). Put presence
   requirements in `binding:"required"`. Do **not** add `binding:"required_if=…"`
   to express a domain rule the downstream service already owns.
2. **Document conditional rules in prose** — in the field's doc comment and the
   handler's `@Description`. Never as `if/then`/`oneOf`/schema `required`.
3. **Annotate the handler** with swag comments: `@Summary`, `@Description`,
   `@Tags`, `@Accept`/`@Produce`, `@Param`, `@Success`/`@Failure` (use the
   `*Response` / `ErrorResponse` envelope structs), `@Security BearerAuth` for
   protected routes, and `@Router <path> [method]` (path is relative to `@BasePath
   /api`). New tags must be declared with a `@tag.name` line in `cmd/main.go`.
4. **Regenerate & lint:** `make gen && make lint`. Commit the regenerated
   `docs/` artifacts alongside your code change so they never drift.
5. **Add a runnable request** to `test/requests.http` (one per endpoint; include a
   present/omitted pair for any conditional field), seeded from the spec's example
   values.
