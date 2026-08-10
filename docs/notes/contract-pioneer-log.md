# Contract pioneer log

Running record of what we learn while making ADR-0002's contract layer real in fireplace.
Fireplace is the **test bed**: each entry is written so the generalizable part can later be
extracted into the shared skills/templates SSOT (`ai-software-engineering`). Append-only.

---

## 2026-08-04 — Spike: Huma v2 into the existing gin router

**Questions (both answered YES).**

1. Can Huma v2 mount into the api-gateway's existing gin router **inside** the protected
   group, leaving `auth.AuthMiddleware`, legacy routes, and `/swagger` untouched?
2. Do downstream gRPC status codes map cleanly to RFC-7807 problem responses via **one**
   adapter function?

Throwaway branch `spike/huma-gin-mount`, deleted after reading. 14 subtests, all green.

### What worked

**The mount.** `humagin.NewWithGroup(engine, group, cfg)` takes a `*gin.RouterGroup`
directly. Routes registered through it inherit the group's middleware chain, so
`auth.AuthMiddleware` applies to Huma routes with **no changes to the middleware and no
changes to how the group is built**:

```go
protected := api.Group("")
protected.Use(auth.AuthMiddleware())
api := humagin.NewWithGroup(engine, protected, cfg)   // ← the whole mount
```

Proven in the harness: Huma route without a token → **401**; with a valid token → **200**
and a typed body. A legacy gin handler sitting on the same group, the public
`/api/users/signin` route, and `/swagger/*any` all behaved exactly as before. Huma emits
**OpenAPI 3.1** for the wrapped route (`openapi: 3`, operation id, path) — the whole point.

**Coexistence is real.** Legacy swaggo routes and Huma routes live on the same engine and
the same group at the same time. This is what makes ADR-0002's *serialize-on-touch*
adoption policy mechanically possible rather than aspirational.

### The adapter shape

**One function, and it does no mapping of its own** — that's the finding. `apierr.StatusFor`
already owns the gRPC-code → (HTTP status, client-safe message) decision for the entire
gateway. The adapter only *re-serializes* an existing decision:

```go
func problemFor(op string, err error) huma.StatusError {
	code, msg := apierr.StatusFor(err)   // existing gateway-wide mapping
	return huma.NewError(code, op+": "+msg)
}
```

Huma renders the returned `StatusError` as `application/problem+json` automatically —
`type`, `title` (defaulted from status text), `status`, `detail`. No custom error model, no
`huma.NewError` override, no per-handler status switch.

Verified end-to-end for `NotFound→404`, `AlreadyExists→409`, `InvalidArgument→400`,
`Unauthenticated→401`, `PermissionDenied→403`, `Internal→500`, `Unavailable→500` — each
with `Content-Type: application/problem+json` and a matching `status` field in the body.
Also asserted the **downstream message does not leak** (`"plan gone"`, `"boom"` never
appear in the response); the client-safe message from `StatusFor` is what ships.

> Because the existing `apierr` package already centralized this, the RFC-7807 migration is
> a **serialization change, not a semantics change**. That is why ADR-0002 can call the
> `problem+json` body a known, deliberate break and stop there.

### Gotchas found (these are the real value)

**1. Identity does not cross into Huma handlers.** `humagin`'s `Context()` returns
`c.Request.Context()`, **not** the `*gin.Context`. The existing middleware publishes
identity via `c.Set("userId", …)` — gin's own KV store — which a Huma handler therefore
cannot see. Needs a one-time shim, mounted after `AuthMiddleware` and before any Huma
route:

```go
func BridgeIdentity() gin.HandlerFunc {
	return func(c *gin.Context) {
		if userID, err := auth.GetUserID(c); err == nil {
			c.Request = c.Request.WithContext(
				context.WithValue(c.Request.Context(), ctxKey{}, userID))
		}
		c.Next()
	}
}
```

This is written once for the gateway, not per endpoint. Note it interacts with **ADR-0001**:
when identity moves to gRPC-metadata-and-context, this bridge is where the two meet — worth
revisiting so we don't end up with two identity seams.

**2. All Huma config paths are relative to the mount group.** Mounting on a group with base
path `/api` and setting `DocsPath: "/api/docs"` produces **`/api/api/docs`**. Everything
gets the prefix — `SchemasPath` default `/schemas` landed at `/api/schemas`. Set these
paths *relative to the group*, or mount the doc surface separately.

**3. Docs and spec end up behind auth.** Because the mount is on the protected group,
`/…/docs` and `/…/openapi.yaml` inherit `AuthMiddleware` and return 401 without a token.
Legacy `/swagger` is public by design ("browsable in dev"). This is a **decision to make,
not a bug**: either accept an authenticated spec surface, or mount Huma's docs on the
public group while keeping operations on the protected one.

### Dependency cost (material, and forced)

`huma v2.39.1` requires **`gin v1.12.0`** and **Go 1.25.0**. Confirmed from the module
graph — `huma/v2@v2.39.1 → gin@v1.12.0` — so MVS raises fireplace's floor; pinning gin back
to 1.10.0 does not take effect while huma is a dependency.

| | before | after |
|---|---|---|
| gin | v1.10.0 | **v1.12.0** |
| go directive | 1.24.2 | **1.25.0** (also bumps `go.work`) |
| new indirect surface | — | `quic-go` + `qpack` (**via gin 1.12's http3**, not huma), `goccy/go-yaml`, `bytedance/gopkg` |

`quic-go` reaches the build graph (24 packages) through **gin**, not huma —
worth stating plainly so it isn't misattributed later. `mongo-driver/v2` appears in `go.mod`
after `tidy` but `go mod why` reports the main module does not need it.

The whole gateway **built and its full test suite passed** on gin 1.12 with no code changes.
Still, this is a framework-version bump across every gin handler in the service, so it
belongs in the first serialize-on-touch slice's risk note rather than being discovered mid-feature.

**Repo hygiene, noticed in passing:** `services/api-gateway/go.sum` is **gitignored**. That
undercuts reproducible builds and specifically weakens ADR-0002's CI *regenerate-and-diff*
gate, which assumes a pinned, verifiable module graph. Worth fixing before the gate is built.

### Template-worthy (candidates for extraction to the SSOT)

Nothing here should be extracted yet — one spike is one data point. Flagged for when a real
slice confirms them:

- **The adapter is a two-liner *if* the service already has a single error-mapping boundary.**
  The generalizable rule isn't "write `problemFor`", it's **"a service needs one
  `StatusFor`-shaped seam before it can serialize its contract."** Fireplace had one;
  a repo without one pays that cost first. Good candidate for a `setup`/adoption precondition.
- **The identity bridge is a named, recurring shape** — any framework-adapter mount
  (huma-on-gin, huma-on-chi) has to answer "how does middleware-set identity reach the typed
  handler." Worth a documented pattern, not a per-repo rediscovery.
- **"Config paths are relative to the mount point"** — a footgun that costs 20 minutes each
  time. One line in whatever contract template we ship.
- **The dependency floor check** (`go mod graph | grep <lib>`) belongs in the adoption
  checklist: *what version floor does the contract library impose, and does the service
  survive it?* Answer before slice ⓪, not during.

### Not proven

The harness mirrors `SetupRouter`'s **shape** (engine → `/swagger` → `/api` → public →
protected + `AuthMiddleware`), not the fully-wired production router, which needs a DB and
Consul. The mount mechanism and middleware inheritance are proven against the **real**
`auth.AuthMiddleware`; wiring into the real `SetupRouter` is still unproven and is the
first thing slice ⓪ should do.

Also unproven: `oasdiff`, Spectral, `openapi-typescript`/`openapi-fetch`, CI
regenerate-and-diff, and the buf plane — none were touched.

---

## 2026-08-06 — Planning FS-0002: three extraction findings

No code yet. The validation run has already paid for itself twice over in findings, which is
the point of running it on an already-shipped feature.

### 1. Gate tooling is template-land, not feature-land (the strongest finding so far)

The contract gates — `oasdiff` + `.oasdiff.yaml` allowlist, `.spectral.yaml`,
regenerate-and-diff CI, `make openapi` / `make client` — are **ADR-0002 implementation**, not
a product capability. They were deliberately kept out of FS-0002 (its Out of Scope stands)
and tracked as a side task anchored to the ADR (#46) rather than given a feature spec.

**Every artifact that task produces is extraction-bound for `setup/templates`:**
`contract.yml`, `.oasdiff.yaml`, `.spectral.yaml`, and the Makefile targets.

> In template-land this entire task disappears into **"setup copies these files."**
> That is exactly why it never deserved an FS — there is no behavior to specify, only
> infrastructure to place. A capability answers *what can the system do*; this answers
> *what must be present before the system can be governed at all*, which is `setup`'s
> question, not `scope-it`'s.

The generalizable rule: **if the deliverable is a file that every repo needs identically, it
is a template, not a feature.** Feature specs are for things that differ per repo.

### 2. `Implements ADR-NNNN` is a second legitimate anchor form

Infrastructure tasks have no FS, so the tracker anchor has to be the ADR. This is a real
second anchor type alongside `Implements FS-NNNN §<section>`, not a workaround.

**Confirmed the current skills would reject it.** `develop`'s entry gate says *"No FS
reference? Stop and flag it — do not implement unanchored work,"* and `code-review`'s spec
axis resolves only `Implements FS-NNNN`, scoring anything else unanchored (MED). So #46 as
written would be refused by `develop` and dinged by `code-review`.

→ **SSOT skill patch required** (not a fireplace fix): accept ADR-implementation as an anchor
type for infrastructure tasks in `develop`'s entry gate and `code-review`'s spec axis. The
distinction to encode: an ADR-anchored task is judged against the ADR's *constraints*, not
against acceptance criteria, because there is no FS to carry them.

### 3. The thin-line format assumes greenfield; brownfield needs two lines

Serialize-on-touch creates a work order against a capability that **already ships**. One
checkbox cannot express two independent states — *does the capability exist for users?* (yes)
and *is its work order done?* (no). Resolved with the **two-line serialize-on-touch rule**:

```
- [x] Profile view and edit → FS-none                  (pre-existing behavior)
- [ ] Typed (serialized) profile surface → FS-0002     (this feature's work)
```

The FS's checkbox flips only when its own acceptance criteria ship; the capability being live
does not check it. `FS-none` is a legal pointer for pre-existing behavior with no work order
(ADR-0002 already used the convention).

**This is the highest-value extraction candidate in the log**, because brownfield adoption is
the *common* case — every repo adopting this system on an existing codebase hits it
immediately, via `spec-bootstrap`'s retroactive FSs if not via serialize-on-touch. The format
authority currently describes only one line per capability and says nothing about a retrofit
line pairing with a shipped-capability line.

→ **Two `docs/specs/README.md` edits now outstanding**, both of which then re-copy to the SSOT
template `docs-specs-README.md` as a deliberate act:
1. `§API surface` tables carry `status · code` per error row (ADR-0004).
2. The two-line serialize-on-touch rule, including `FS-none` as a legal pointer.

Until (1) lands, FS-0002's table is an instance of a format its own authority does not
describe. Until (2) lands, `spec-audit` has no rule saying `FS-none` is legal and may flag it.

### Amendment to the 2026-08-04 entry

That entry reports the gRPC→problem adapter as a **two-liner**. That was true *against the
stock `huma.ErrorModel`*. ADR-0004 adds a `code` extension, and `huma.ErrorModel` has no
extension field — so the adapter becomes a custom error type (implementing `GetStatus()`,
`Error()`, **and `ContentType()`**) plus a `huma.NewError` override. Still **one seam**, which
was the load-bearing claim; but not two lines.

The `ContentType()` requirement is the trap worth repeating: it is what emits
`application/problem+json`, and a custom model that omits it silently degrades to
`application/json` while passing any test that asserts only status and body.

---

## 2026-08-06 — #46 implementation: what building the gates actually taught

### ADRs guess tool interfaces; implementations correct them

ADR-0002 specifies "deliberate breaks go through an **`.oasdiff.yaml`** allowlist." oasdiff
has no YAML allowlist format — its native mechanism is `--err-ignore <file>` taking a
**plain-text** file, one lowercased substring match per line. Implemented as
`.oasdiff-ignore`.

The *decision* the ADR made (deliberate breaks pass only via a reviewed allowlist) is
untouched; only the filename and syntax were wrong. This is the expected division of labour —
an ADR written before the toolchain is exercised will guess at interfaces, and the
implementation corrects the mechanism without reopening the decision. **Do not amend the ADR
for this**; it is not a changed decision.

**Extraction consequence:** the template that eventually ships must carry the *verified*
filename, not the ADR's guess. Any doc generated from the ADR text alone would propagate the
error to every future repo.

### `make` runs each recipe line in its own shell — and it produced two false greens

The guard pattern `@if [ ! -f spec ]; then echo SKIPPED; exit 0; fi` followed by real work on
the next line **does not work**: `exit 0` ends only that line's shell, and make proceeds.
Both failures were silent-success, the exact class the gates exist to prevent:

1. `make openapi`'s `> openapi.yaml` redirect **created an empty spec file** even though the
   generator did not exist. That empty file then satisfied every downstream `[ -f ]` guard.
2. With the empty file present, `make openapi-breaking` diffed two empty specs and reported
   **"No changes detected"** — a passing breaking-change gate on a repo with no contract.

Fixed by making each recipe a single `if/else` shell block. Verified by exit code per target,
not by reading output.

> **This is the strongest argument yet for the "no silent skips" acceptance criterion.** The
> criterion was written as a nicety; it caught two real false greens within an hour. A gate
> that cannot distinguish *passed* from *did not run* is worse than no gate, because it emits
> a green check either way. Any extracted template must keep the skip-with-reason pattern and
> the per-target exit-code assertion.

### A ruleset is unverified until it has rejected something

`.spectral.yaml` was written, then tested against three fixture specs rather than assumed
correct. This immediately caught an invented rule: **`operation-4xx-response` is not a
`spectral:oas` rule** (it exists in other rulesets), so the ruleset failed to load at all —
`Cannot extend non-existing rule`. Reimplemented as an explicit custom rule.

Verified: bad spec → 3 errors, rc=1 · problem+json-without-`code` → the code rule fires, rc=1
· compliant spec → rc=0. All four enforced rules have now rejected something.

**Extraction rule: ship rulesets with their fixtures.** A `.spectral.yaml` copied into a new
repo by `setup` is unverified config until it has failed a known-bad spec there too. The
fixtures are part of the template, not scaffolding to discard.

### The gate task cannot close its own acceptance criteria

#46 stands up oasdiff, regenerate-and-diff, and the client-staleness check — none of which can
be *proven* until `openapi.yaml` exists (FS-0002 slice 3). Three of its seven acceptance
criteria are **wired but unproven**, by construction, not by omission.

This is inherent to gate tooling and will recur in template-land: `setup` copying these files
into a fresh repo produces the same unproven state, and it stays unproven until that repo's
first serialized endpoint. Worth stating in whatever doc ships with the template, so the gap
reads as expected rather than as a broken install.

### `.gitignore` hygiene is a contract concern, not housekeeping

`go.sum` was ignored repo-wide (bare `go.sum` at `.gitignore:37`, **zero** lockfiles tracked
across 8 modules). Un-ignored and committed. A regenerate-and-diff gate assumes a verifiable
module graph; a monorepo where no module is pinned cannot make that promise, so this was
blocking rather than adjacent. `go mod tidy` also reclassified one dependency
(insights-service protobuf, indirect → direct, same version) — a correction, not a bump.

---

## 2026-08-07 — Validation policy: the rule that generalises

Implementing FS-0002 forced a decision ADR-0002 never anticipated, and the reasoning is the
most portable thing this run has produced. Recorded as **ADR-0005**.

### The rule

> **Reject unknown input when the consumer ships with you. Tolerate it when it does not.**

Strictness is not a taste question, it is a function of the **deployment model**. That single
sentence resolves what looked like an inconsistency between the two planes:

| Plane | Consumer | Policy | Enforced by |
|---|---|---|---|
| 1 — HTTP | first-party web client, same release | **strict** (`additionalProperties:false`, 422) | huma, at the boundary |
| 2 — gRPC | services, independently deployed | **tolerant** (protobuf ignores unknown fields) | `buf breaking`, at CI |

Same goal — no silent drift — enforced where each plane can actually enforce it. Plane 2 gets
protobuf's rolling-upgrade property, which is the reason to use protobuf at all.

### The argument that decided it

We tried tolerant-on-plane-1 first, on forward-compatibility grounds, and reverted. The
deciding case:

```
PATCH {"biio": "my new bio"}   ->  200 OK, profile unchanged
```

On a PATCH, ignoring an unknown member turns a client typo into a **silent no-op**. The user
believes they saved something they did not. Forward compatibility is the stronger concern only
when clients and servers deploy separately — for a client that ships in the same release and
consumes generated types, it is near-worthless, because that client *cannot* emit an
undeclared field by accident.

**Generalisable form: strictness moved left.** Once a repo generates a typed client, runtime
strictness stops being the typo-catcher (the compiler is) and becomes purely a drift alarm.
That changes the cost/benefit, and it only holds in repos that actually generate a client — so
it belongs in the template as a *conditional* rule, not an absolute one.

### The other half: two layers, two statuses

| Layer | Question | Where | Status |
|---|---|---|---|
| shape | is this a well-formed request for this op? | boundary (huma, from the type) | **422** |
| domain | is it allowed by the domain's rules? | the **owning** service | **400** + domain code |

The gateway never restates a downstream rule — one place per rule, so neither can drift. A
client can tell "you sent garbage" from "you sent something valid that we refused", which the
usual single-400-for-everything cannot express.

### Request-type design rules (learned, not assumed)

- **Read-only fields never appear in a request type.** `id`/`createdAt`/`updatedAt` are real
  profile fields, but a request type is not a resource — it is the set of *writable* fields.
  This makes a class of mass-assignment bug unrepresentable rather than defended against.
- **Identity is never a body field** (ADR-0001). `id` absent from `UpdateProfileRequest` is a
  security property, not an omission.
- **`omitempty` is load-bearing in huma.** A field is REQUIRED unless its json tag has it. We
  shipped `required: [name, displayName, bio]` by accident, which would have rejected the empty
  `{}` body the spec calls a valid no-op. Any extracted template must call this out — it is
  invisible until something rejects a legal request.

### Extraction targets

- `docs/agents/contract.md` now carries `plane1_request_strictness` / `plane2_request_strictness`
  plus the request-type rules, so a generic skill can read the policy instead of assuming one.
- `docs/specs/README.md` needs a third outstanding edit: the `§API surface` guidance should say
  that request-body rows list **writable fields only**, and that 422-vs-400 is the shape/domain
  split rather than a free choice per endpoint.
- **Ship the revisit trigger with the rule.** "Strict" is correct only while the client ships
  with the server; a template that states the rule without its precondition will be
  cargo-culted into a repo with third-party consumers, where it is wrong.

---

## 2026-08-09 — Pre-extraction verification: the alarm check, and what it broke

Ran the full mesh verification before extracting anything to the SSOT. Most of it is green.
The two failures are both in the **breaking-change gate**, and one of them invalidates a file
that was about to be templated.

### The break-test had never been run — and the allowlist does not work

This is the finding. `.oasdiff-ignore` documents its own format as:

> "Each line is matched (lowercased, substring) against the breaking-change message oasdiff
> prints. Run `oasdiff breaking <base> <revision>` to see the exact text, then paste it here."

**Following that instruction produces an allowlist that suppresses nothing.** Observed
directly: base = committed `openapi.yaml`, revision = same spec with `displayName` renamed.

```
oasdiff breaking base.yaml rev.yaml --err-ignore .oasdiff-ignore --fail-on ERR
→ 2 errors, exit 1     [response-required-property-removed]
    "removed the required property `displayName` from the response with the `200` status"
```

Pasting that message verbatim into the ignore file, exactly as the file instructs — **still
exit 1**. The gate stays red. The working format requires the ignore line to carry the
**method and path as well as the message**, all lowercased, on one line:

```
in api get /users/profile removed the required property `displayname` from the response with the `200` status
in api patch /users/profile removed the required property `displayname` from the response with the `200` status
→ 0 errors, exit 0
```

One line per **(method, path, message) triple** — not one per message. A break affecting two
operations needs two lines.

> **This is "ADRs guess tool interfaces" recurring one level down.** The 2026-08-06 entry
> caught ADR-0002 guessing the *filename*. The implementation that corrected it then guessed
> the *matching semantics* — and wrote the guess into the file as authoritative documentation.
> A correction is not automatically verified just because it corrected something.

The failure mode is at least **fail-closed**: an unrecognised entry blocks a deliberate break
rather than admitting an accidental one. But the practical consequence is worse than it looks —
the first engineer with a legitimate break finds the documented escape hatch doesn't work, and
the tempting fix is to weaken `--fail-on` or drop the gate.

**Extraction consequence: the `.oasdiff-ignore` starter is NOT extractable as written.** It
must ship with the verified triple format and, per the rule below, with a fixture proving a
listed break is actually suppressed.

### The gate has never run against a real baseline

`make openapi-breaking` resolves its baseline from `origin/main:services/api-gateway/openapi.yaml`.
That file does not exist on `origin/main` — the whole contract layer is still on
`feat/46-contract-gate-tooling`. So the target takes its skip branch:

```
SKIPPED: no baseline spec on origin/main yet (first serialization)   rc=0
```

Correct behaviour, honestly reported (the no-silent-skips criterion working as designed), but
it means **the CI breaking gate has never executed a real comparison** — only the synthetic
one above, run by hand. It cannot until this branch merges.

### The first serialization's breaks are invisible to oasdiff — by construction

`.oasdiff-ignore` predicted two entries would be needed once slices 3–4 landed: envelope
removal and the error-shape change. The slices landed; **neither entry is needed, and adding
them would be wrong.**

oasdiff compares `openapi.yaml` against its own previous revision. Both of those breaks are
relative to the *legacy swaggo `docs/swagger.json`* — a different document, in a different
OpenAPI version, that oasdiff never sees. They are **cross-document** breaks, and plane-1
breaking detection is structurally blind to them.

> **Generalisable, and it belongs in the template.** Under serialize-on-touch, an endpoint's
> *first* serialization is the one change whose breaks the breaking-change gate cannot catch.
> The gate governs everything after. Whatever ships must say this out loud, or every adopting
> repo will believe slice ⓪ was checked when it was not — and slice ⓪ is exactly where the
> deliberate breaks live.

The empty allowlist is therefore **correct as-is**, but for a reason its own comment gets
wrong: not "the endpoints aren't serialized yet" (they are), but "these breaks are not of a
kind oasdiff can see."

### `oasdiff@latest` is not reproducible, and moves the Go floor

`OASDIFF = go run github.com/oasdiff/oasdiff@latest`. Resolved today to **v1.28.0, which
requires go >= 1.26** and silently triggered `switching to go1.26.5` — against a service whose
`go.mod` directive is 1.25.0 and whose CI pins `go-version-file: go.mod`. A gate whose tool
version floats is a gate that can turn red on a day nobody changed the contract. Pin it.

### Everything else verified green

- **Spec is genuinely derived.** `make openapi` is deterministic and reproduces the working-tree
  `openapi.yaml` byte-for-byte (hash-compared across two runs). *Caveat: derived-and-current is
  proven for the **working tree**; the round is still uncommitted, so "committed = derived"
  is not yet true on any branch.*
- **Spectral passes** on the committed yaml (`rc=0`, no errors).
- **Client is derived and sole.** `make client` reproduces `schema.d.ts` identically. Every
  `/users/profile` call routes through `src/api/profile.ts` on the generated client; the only
  remaining hand-written `fetch` calls are signin/signup, which are unserialized. `tsc` is
  clean across all contract-touched paths (26 pre-existing errors survive in `Todo.tsx`,
  `NotesContext.tsx`, `notesService.ts` — untouched by this run, unrelated debt).
- **Docs cohere.** FS-0002 is `Status: shipped`; `SPECIFICATION.md ## Users` carries the
  two-line split with the `FS-none` line intact and the serialization line now `[x] → FS-0002`.
  ADR-0002/3/4 are present and uncontradicted: no `@`-annotations on the typed handlers, no
  `models.User` in the transport path, error model matches ADR-0004.
- **`errors` required is deliberate, not the `omitempty` trap recurring.** `Problem.Errors`
  drops `omitempty` on purpose (FS-0002 R16: present-and-empty so the FE never null-checks),
  and the schema's `required: [code, errors]` is the faithful consequence. Worth naming because
  it looks identical to the bug the 2026-08-07 entry documented, and is its exact inverse.

### Two rules this run made, that the log had already written down and the repo did not follow

1. **"Ship rulesets with their fixtures."** The 2026-08-06 entry established this after
   Spectral failed to load an invented rule. **The fixtures were never committed** — no
   fixture file exists anywhere in the repo or its history. The rule was recorded and then
   not applied to the very ruleset that produced it.
2. **The three flagged skill patches were never made.** `Implements ADR-NNNN` as an anchor
   type (`develop` entry gate, `code-review` spec axis), the brownfield two-line rule with
   `FS-none`, and the `status · code` Errors-column format are all still absent from the SSOT —
   verified by grep across `.claude/skills/`. Only `walk-it` was added during this run, and it
   carries no fireplace-specific wording, so no leakage occurred. Nothing leaked *because
   nothing was patched.*

> The pattern worth extracting is the meta one: **a pioneer log records findings but does not
> apply them.** Both gaps above are places where the finding was written down correctly and
> the follow-through silently didn't happen. Extraction is where that debt comes due — which
> is an argument for running this verification pass *before* every extraction, not once.

---

## 2026-08-09 — Fixes, re-verification, and extraction. This log's job is done.

The failures from the entry above are closed, Phase 1 re-ran green, and the layer is extracted
to the SSOT. Final entry: fireplace's role as the test bed ends here.

### The go-floor decision (surfaced, not assumed)

Pinning oasdiff forced a real choice. **Every oasdiff release back to v1.18.6 declares
`go 1.26`** — checked against the module proxy, not assumed — while this module's directive is
`1.25.0` and `go.work` pins all eight modules to it. There was no "pin to an older compatible
version" option; that path does not exist.

Resolved by **installing a pinned prebuilt binary** (`oasdiff_1.28.0_<os>_<arch>` from GitHub
releases) into a gitignored `.tools/`, rather than `go run`. `go.mod` and `go.work` are
untouched.

> **The generalisable form, and why it beat the alternatives:** gate tooling moves faster than
> the service it guards. Running it via `go run` silently drags in a second toolchain whenever
> its floor exceeds yours — making a *gate* depend on a network toolchain fetch, and breaking
> under `GOTOOLCHAIN=off`. Bumping the whole monorepo to 1.26 to satisfy a lint tool is the
> tail wagging the dog. **A pinned binary keeps the gate's toolchain independent of the
> service's** — which is also the property that lets repos on different Go versions share one
> gate config, and therefore the property the template needed.

### Fixtures: minimum viable, and they earned their keep immediately

`contract-fixtures/` now holds one known-bad Spectral spec (trips three custom rules at once —
`3 errors, rc=1`) and one oasdiff break pair with a worked allowlist.

The allowlist file does double duty: it is the fixture's expected-pass input **and** the
reference for the entry format. That matters because the format is the thing the previous
entry proved was wrong. Verified in both directions against the pinned binary:

```
unallowlisted break  → rc=1     (the gate can still reject)
allowlisted break    → rc=0     (the escape hatch actually works)
```

`.oasdiff-ignore` now documents the **verified triple format** — one line per
(method, path, message), lowercased — with a pointer to the working example, and explains why
it is **correctly empty**: the first serialization's breaks are cross-document and invisible to
oasdiff, so pre-populating it would create entries that match nothing and rot.

**Deferred deliberately** (not forgotten, and recorded here as the scope trim it was): a
per-rule fixture suite, and a `make gates-selftest` target wired into CI. Today the fixtures
verify the gates at *adoption* time, not continuously. That is the next hardening step, and it
was left out of the template too — so the template matches what has actually been exercised
rather than what would be nice.

### What extracted, and the one rule that decided each case

The rule from the 2026-08-06 entry held up under contact: **if the deliverable is a file every
repo needs identically, it is a template; feature specs are for what differs per repo.** Nine
template files, one interview question, four skill patches.

What stayed fireplace-local is as informative as what left: the gin/humagin adapter choice,
swaggo coexistence, the `go.sum` un-ignoring, `PROFILE_NAME_EMPTY`, and the ADR files
themselves. **Fireplace's ADRs were not moved or edited** — they are immutable records of what
*this* repo decided. The template ADR is a fresh document a future repo fills and commits as
its own; it merges what fireplace learned across four ADRs into one decision, because a repo
adopting the whole layer at once decided it at once, and the record should not imitate a
history it did not have.

The stack-specific patterns (error adapter, identity bridge, the problem-emitting auth fork,
`ContentType()`, `omitempty`) extracted as **pattern notes, not copy-paste code** — each names
the seam and the trap, and says explicitly to re-derive the code. Every one of them cost real
debugging time here; that is the entire reason they are worth carrying.

### The honest closing state

- **Everything is still uncommitted** on `feat/46-contract-gate-tooling`. "Committed = derived"
  is proven for the working tree, not for any branch. It becomes true on commit.
- **The breaking gate has still never run against a real baseline** and cannot until this
  branch merges — `SKIPPED: no baseline` is correct, not broken. The synthetic proof above is
  what stands in until then.
- **Plane 2 remains unwired.** `docs/agents/contract.md` lists it explicitly as a gap rather
  than omitting it, so it stays auditable.

### Why this log ends here

Fireplace was the test bed, and the test is over: the layer shipped, was verified against
evidence rather than memory, and the generalizable part is in the SSOT. Anything learned from
here on is **fireplace's own maintenance**, not pioneering — it belongs in the ADRs and the
commit history like any other work.

The real validation is no longer in this repo. **quanta adopting the layer through `setup` is
the template's first consumer test** — the first time this content meets a codebase that did
not grow it, and the only way to find what is still accidentally fireplace-shaped. Extraction
from the originating repo cannot reveal that, by construction.

One last finding, from the run rather than the code: **the log recorded findings faithfully and
applied none of them.** Every gap the verification pass found — missing fixtures, unmade skill
patches, an unverified allowlist format — had already been written down here, correctly, and
then not done. A log is a memory, not a mechanism. The mechanism is running the verification
pass *before* extraction, every time, and treating "we wrote that down" as evidence of nothing.

### Addendum, same day: the last two caveats are closed

The section above was written while the work was still uncommitted. Both of its standing
caveats are now resolved, so they are corrected here rather than left to mislead.

FS-0002 and the gate task were committed and merged to `main`, and `main` was pushed. That
put `openapi.yaml` on `origin/main` — **the baseline the breaking gate had never had.** Run
against it for the first time:

```
make openapi-breaking          → "No changes detected"                    rc=0
# same gate, spec mutated (displayName -> display_name):
make openapi-breaking          → 2 errors [response-required-property-removed]  rc≠0
```

Clean when clean, red when broken, **against the real committed baseline** — not a synthetic
fixture pair. The ratchet is live from this merge onward, exactly as the design predicted.

So: *committed = derived* is now true on `main`, and the gate's "never run for real" caveat is
retired. What remains open is unchanged and unaffected — plane 2 is still unwired, and the
`gates-selftest` target is still the named next hardening step.

The slice-⓪ blind spot is **not** closed by this and never will be: it is structural. The
breaks FS-0002 actually shipped (envelope removal, error-shape change) were relative to the
legacy swaggo document and remain invisible to this gate. What just became guarded is
everything *after* this point.

---

## 2026-08-10 — The extraction was endpoint-complete and middle-empty

The entry above declared this log finished. It was wrong, and the way it was wrong is the
finding.

### What was actually extracted

Auditing the fleet for the concepts this run produced showed them landing in exactly two
places: where work is **judged** (`develop`, `code-review`, `spec-audit`) and where repos are
**scaffolded** (`setup` + templates). Every skill in the middle — where work is **planned** —
knew nothing:

| Skill | Missing |
|---|---|
| `scope-it` | no notion of an API surface at all; nothing about serialize-on-touch |
| `spec-to-issues` | required `§API surface` in slice references, but had no concept of **slice ⓪** |
| `spec-update` | writes thin lines and flips `shipped` — the exact place retrofit pairs are authored — with no two-line rule |
| `spec-bootstrap` | reverse-engineers lines from built code and never mentioned `FS-none` |

`write-a-spec` was the exception, and instructively so: it **points at `docs/specs/README.md`
as format authority** rather than restating the table rules, so the three authority edits
reached it for free. That is the "point at the authority, never embed it" rule paying for
itself — one edit to the authority updated a skill nobody touched.

### Why it happened, and why it was invisible

**The planning knowledge lived in the humans during the run.** Nobody needed a skill to say
"wrap the endpoint first" — we were the ones who decided it, in this repo, in this thread. The
gates got patched because gates *fail loudly* when they lack a rule; planning skills fail
**silently**, by simply not raising something. An unasked question leaves no trace.

That is the generalisable trap: **extraction naturally follows the failures you felt.** Every
gap found here was in a skill that had never blocked us, because it could not. A verification
pass that only re-runs the gates will never surface it — the gates were green the whole time.

### The four patches

Each is 2–4 lines and **points at an authority** (the contract ADR, `docs/specs/README.md`)
rather than restating it — same discipline that saved `write-a-spec`:

- **`scope-it`** — an *API surface* exploration area (does this add/change endpoints; if they
  exist unserialized, flag serialize-on-touch in the draft notes), plus the two-line brownfield
  invariant in the locking rules.
- **`spec-to-issues`** — legacy endpoints ⇒ the first slice is **slice ⓪**, the wrap. The FE
  cutover is its **own late slice** (where the grandfather clause executes). Deliberate shape
  breaks ride ⓪ or the cutover and are **verified manually**, because the ratchet cannot see
  breaks against the pre-serialization world.
- **`spec-update`** — author the retrofit **pair**; the new line's checkbox flips only on its
  own acceptance criteria, and live legacy behavior never checks it.
- **`spec-bootstrap`** — `FS-none` is now its **default pointer**, and its lines are `[x]`:
  these capabilities ship today, which is the entire premise of reverse-engineering them.

The last one closed an orphan nobody had noticed: `spec-bootstrap` had been emitting `FS-TBD`,
a placeholder **no other skill in the fleet consumed or resolved** — not `write-a-spec`, which
was supposed to, and not `spec-audit`, which would have flagged it. One term, one writer, zero
readers, quietly rotting in the one skill that runs on repos with no specs yet.

### The rule this run leaves behind

> **Extraction is not done when the gates are green. It is done when every verb in the
> lifecycle can state the rule without a human in the room.**

Judging skills are the *easy* half — they are exercised constantly and complain when
underspecified. Planning skills are where tacit knowledge hides, because their failure mode is
an absence. Audit them by concept, not by whether anything broke: for each idea the run
produced, grep the fleet and ask *which verb raises this, and when?* A concept that resolves
only to skills that **check** work is a concept the system cannot **plan** with.

Verified after patching: `slice ⓪`, `serialize-on-touch`, `FS-none`, and the two-line pair each
now resolve to at least one planning skill, not only to the gates.

---

## 2026-08-10 — First fleet-adoption pass: the new machinery caught things on its first outing

Ran the upgrade against barrowspire and quanta — both older-vintage repos, both pre-contract-layer.
The interesting part is not the upgrade; it is that two pieces of machinery added *during* this
run earned their keep immediately, on the first repo they touched.

### The Class column caught its first false positive

`setup`'s manifest gained a permanent column splitting every piece into **tracked copy**
(canonical content the repo never edits — byte-matches its template, safe to re-copy) versus
**copy-once** (forks at birth and diverges on purpose — additive patches only, never re-copy).

The audit immediately flagged `docs/agents/tracker.md` as `STALE (7 vs 15 lines)`. It is not
stale. It is **copy-once**, and it forked correctly at birth when one mode block was uncommented
and the rest deleted. A line-count diff cannot tell "behind the template" from "correctly
diverged" — only the classification can.

> **The rule that generalises: staleness is only a meaningful question for files that were
> supposed to stay identical.** Ask it of a living file and you get a false positive every time,
> which trains people to ignore the check. This is why the column had to be *permanent manifest
> metadata* rather than a judgement call at audit time.

The test for the column is **not** "does it have placeholders." `tracker.md` and `.oasdiff-ignore`
contain no `{PLACEHOLDER}` at all, yet both accumulate repo state — a chosen mode, reviewed
allowlist entries. The real test is: **does repo-specific state accumulate in this file after it
is copied?**

### The STOP rule caught three template gaps before a live run hit them

`setup`'s "template missing → STOP, do not invent" rule fired on three manifest pieces that had
**no template at all**: `docs/adr/README.md`, `docs/issues/README.md`, `.audit/README.md`.

All three already existed in consumer repos — which means they had been **hand-authored**, at
different times, in different words. Exactly the drift the copier rewrite was built to prevent,
sitting in the blind spot where the manifest listed a piece the templates could not supply.

Authoring them revealed the cost of having let it run: the three hand-written ADR READMEs had
each independently grown something the others lacked — one had the capability-vs-decision
tiebreaker, one had "check ADRs before architectural changes," one had named itself the schema
authority. **No single copy was the best copy.** The template merges all three rather than
picking a winner, because picking one would have silently discarded two good rules.

> Conventions do not drift loudly. They drift by each repo's copy being *slightly better* at
> something and slightly worse at everything else, until no two agree and none is canonical.

### What the pass actually did

Both authorities re-copied into barrowspire and quanta (`docs/specs/README.md` 63 → 137,
`docs/agents/README.md` 43 → 75, `docs/adr/README.md` → 43), plus the newly-authored issues and
audit READMEs. Verified in both directions before writing: **zero sections existed only in the
repo copies**, so the re-copy was provably additive in effect — which is the whole justification
for the tracked class.

The upgrade reached the repos *because the skills are symlinked and the docs are not*. All three
repos already had every skill patch from this run, live, with no action — and none of the doc
changes. That asymmetry is the thing to remember: **skills propagate, authorities do not.** A
repo can be running rules its own documents have never heard of, which is precisely the state
barrowspire was in.

### Still open, and honest about it

Fireplace itself is now the **most** stale repo in the fleet on `docs/agents/README.md` — the one
document that carries `contract.md`'s schema — because this run edited the template without
re-copying it here. The pass covered barrowspire and quanta by explicit scope. Fireplace is due
the same treatment and has the strongest claim to it.
