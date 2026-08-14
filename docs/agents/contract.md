# Contract config (read by develop's contract check + code-review's spec axis; written once)
# One contract config per repo. Skills READ this file; they never write it.
#
# WHY THIS FILE EXISTS: the skills are generic and must not hardcode a toolchain.
# `develop`'s pre-flight says "regenerate the OpenAPI spec and the TS client" without
# naming a command; this file is where a repo answers that. Same role tracker.md plays
# for issue backends: backend-neutral skill + per-repo binding.

# --- Plane 1: client <-> api-gateway (HTTP) ---
plane1_spec: services/api-gateway/openapi.yaml
plane1_regen: make -C services/api-gateway openapi
plane1_client_regen: make -C services/api-gateway client
plane1_client_dir: client/src/api/generated
plane1_lint: .spectral.yaml
plane1_breaking: oasdiff
plane1_breaking_version: 1.28.0        # pinned prebuilt binary in services/api-gateway/.tools/
                                       # NOT `go run @latest`: unpinned gates turn red on days
                                       # nobody touched the contract, and every oasdiff release
                                       # requires go >= 1.26 while this module is on 1.25.0
plane1_breaking_allowlist: .oasdiff-ignore
plane1_gates: make -C services/api-gateway gates
plane1_fixtures: contract-fixtures/    # known-bad inputs the gates must reject + the worked
                                       # allowlist example; run them after any gate config change

# --- Error vocabulary ---
errcode_pkg: common/errcode
# `code` is contract: removing or repurposing one is breaking, adding one is not.
# Domain codes are added when a real failure needs distinguishing — never speculatively.

# --- Plane 2: service <-> service (gRPC) ---
plane2_proto: common/api/proto
plane2_lint: buf lint
plane2_breaking: buf breaking
# NOT YET WIRED — plane 2 governance is untouched by issue #46. Listed so the shape
# is visible and the gap is explicit rather than silently absent.

# --- Validation policy (ADR-0005) — read this before adding an operation ---
# Two layers, two statuses, no overlap:
#   SHAPE  -> the boundary (huma, from the Go type)      -> 422 VALIDATION_FAILED
#   DOMAIN -> the OWNING service, never the gateway      -> 400 + specific code
# The gateway never restates a downstream rule.
plane1_request_strictness: strict     # additionalProperties:false; unknown member -> 422
plane2_request_strictness: tolerant   # protobuf ignores unknown fields by design;
                                      # buf breaking is plane 2's equivalent guard
# Strictness follows the DEPLOYMENT MODEL: reject unknown input when the consumer ships
# with you, tolerate it when it does not. Revisit plane1 the moment a consumer appears
# that deploys independently (mobile app, third party, public API).

# Request-type rules (ADR-0005 §3):
#   - read-only fields (id, createdAt, updatedAt) NEVER appear in a request type
#   - identity comes from the JWT / verified metadata, NEVER from the body
#   - optional fields MUST carry `omitempty` — huma marks a field required without it,
#     which would reject an empty `{}` body
#   - responses may stay closed (additionalProperties:false); a server never validates
#     its own response, so it only documents the exact published shape

# --- Transitional state ---
# There is NO legacy spec. The swaggo/OpenAPI-2.0 surface was removed outright
# (ADR-0006 §5 as amended), so plane1_spec above is the only description of the HTTP
# surface. Endpoints not yet serialized under FS-0004 are reachable but undocumented —
# the document's info.description states this, so it understates coverage rather than
# misrepresenting it. Do not add a second spec path here.
