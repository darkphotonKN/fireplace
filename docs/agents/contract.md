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
plane1_breaking_allowlist: .oasdiff-ignore
plane1_gates: make -C services/api-gateway gates

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
# The legacy swaggo/OpenAPI-2.0 surface still describes every unserialized endpoint and
# is linted separately by `make -C services/api-gateway lint`. Both retire when the last
# endpoint serializes (ADR-0002 §8).
legacy_spec: services/api-gateway/docs/swagger.json
legacy_lint: make -C services/api-gateway lint
