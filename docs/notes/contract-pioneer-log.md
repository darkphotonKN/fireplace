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
