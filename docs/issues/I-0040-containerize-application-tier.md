---
id: I-0040
status: done
implements: ADR-0012
blocked_by: [I-0039]
labels: [enhancement]
title: "ADR-0012: containerize the application tier and replace Consul with static discovery"
---
Implements ADR-0012 §1, §2, §4

## What to Build

The repository has **no Dockerfiles**, and `docker-compose.yml` contains no application services —
compose is infrastructure only, and every Go service runs on the host under `air`. This issue is
the first containerization of the application tier. It is new work, not a config pass.

**1. A Dockerfile per service** — api-gateway, plan-service, insights-service — and compose
entries to run them.

**2. Fix the bind and registration addresses.** Every service currently does:

    registry.Register(ctx, instanceID, serviceName, "localhost:"+grpcAddr)
    listener, err := net.Listen("tcp", "localhost:"+grpcAddr)

A process binding `localhost` inside a container accepts connections from nothing but itself.
Change the listener to `":"+grpcAddr` and register under the container name. **Nothing on the
internal network can talk to anything until this lands** — it is a precondition, not a cleanup.

**3. Replace Consul with a static registry.** Write an environment-backed implementation of the
existing `discovery.Registry` interface (`common/discovery/discovery.go`) that resolves services
by their compose name, and swap `consul.NewRegistry` for it in each `main.go`. Every call site
goes through `discovery.ServiceConnection(ctx, serviceName, registry)`
(`common/discovery/grpc.go:21`), so **no caller changes**, and the gateway's cached
`*grpc.ClientConn` with lazy redial on `Shutdown` must keep working exactly as it does now
(`services/api-gateway/internal/gateway/insights/client.go`). Remove the Consul container and
its volume.

Ports may stay published to the host for local development — locking them down is I-0041.

## Acceptance Criteria

- [ ] Three Dockerfiles, and all three services defined in `docker-compose.yml`
- [ ] `docker compose up` brings the entire stack — services, Postgres, RabbitMQ, Redis — up healthy
- [ ] No service binds or registers on `localhost`
- [ ] Consul container, volume and all `consul.NewRegistry` call sites are gone
- [ ] `discovery.Registry` and `discovery.ServiceConnection` are unchanged; no gRPC client call
      site was modified
- [ ] Cached `ClientConn` behavior is intact — connections are reused, not dialed per RPC
- [ ] The plan ↔ insights event flow works end to end with everything in containers
- [ ] The migration step from I-0039 runs correctly inside the containerized stack

## Blocked By

I-0039 — containerizing against ten Postgres containers and then re-doing it against one is
wasted work; the database layer settles first.

## Spec Reference

ADR-0012 §1 (containerize all three services), §2 (bind and registration fix as a precondition),
§4 (Consul removed in favor of a static registry).
