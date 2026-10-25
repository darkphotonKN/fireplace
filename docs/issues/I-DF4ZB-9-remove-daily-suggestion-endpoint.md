---
id: I-DF4ZB-9
status: done
implements: FS-DF4ZB
blocked_by: [I-DF4ZB-7, I-DF4ZB-8]
labels: [enhancement]
title: "FS-DF4ZB slice 9: remove checklist-suggestion-daily and GenerateDailySuggestions"
---
Implements FS-DF4ZB §Requirements (R55), §API surface

## What to Build

The old daily-suggestion path goes, now that the client has switched (L14).

- Remove `GET /api/insights/checklist-suggestion-daily` (`getDailyChecklistSuggestion`) from
  `services/api-gateway/internal/insights/typed_insights.go` and the gateway insights client.
- Remove `GenerateDailySuggestions` from `insights.proto` (and its request/response messages),
  the insights handler and service, and its three-call generator in
  `internal/ai/generators.go`. Regenerate stubs.
- Regenerate `openapi.yaml` and the generated client. The breaking gate flags the removal: record
  it in the repo-root `.oasdiff-ignore` per `docs/agents/contract.md`, citing FS-DF4ZB.
- Remove `getDailyInsights` from `client/src/api/insights.ts` and any remaining reference.

The single suggestion (`checklist-suggestion`) and videos are out of scope and untouched.

## Acceptance Criteria

- [ ] `GET /api/insights/checklist-suggestion-daily` and `GenerateDailySuggestions` are gone from
      the gateway, the proto, insights-service and the client.
- [ ] The breaking gate's finding is allowlisted citing FS-DF4ZB; `make -C services/api-gateway
      gates` is otherwise green.
- [ ] `checklist-suggestion` and `suggest-videos` still work unchanged.
- [ ] No API path, contract type, proto message or server type name contains "milestone".
- [ ] Tests pass.

## Blocked By

- I-DF4ZB-7 — the client must no longer call the endpoint.
- I-DF4ZB-8 — serializes the shared contract hotspot (proto, `openapi.yaml`, generated client)
  behind I-DF4ZB-6 and the `client/src/api/insights.ts` edit behind I-DF4ZB-8, following the
  FS's suggested order (Today's moves, then removal).

## Expected write set

- `common/api/proto/insights/insights.proto` + regenerated stubs **(hotspot)**
- `services/insights-service/internal/insights/{handler.go,service.go}`,
  `services/insights-service/internal/ai/generators.go`
- `services/api-gateway/internal/insights/typed_insights.go` (+ test),
  `services/api-gateway/internal/gateway/insights/client.go`
- `services/api-gateway/openapi.yaml` + `client/src/api/generated/` **(hotspot, regenerated)**
- `.oasdiff-ignore` (repo root)
- `client/src/api/insights.ts`

## Spec Reference

FS-DF4ZB §Requirements R55; §Decisions L14; §API surface (removed operation; plane 2 loses
`GenerateDailySuggestions`); §Acceptance Criteria (removal, allowlist, no "milestone" names);
§Out of Scope (single suggestion, videos).

## TDD Approach

- RED: a gateway route test asserting the old path is no longer registered (404 from the router).
- GREEN: remove the handler and regenerate.
- Then: gates green with the allowlist entry.
