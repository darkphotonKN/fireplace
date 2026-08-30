# ADR-0012 — Single VPS with a containerized application tier behind Caddy

Status: accepted
Date: 2026-08-30
Scope: root — governs deployment topology, host port exposure, service discovery, secrets and backups
Related: ADR-0009 (the three services deployed here), ADR-0010 (the database instance this box
runs), ADR-0011 (what was removed to fit)

## Context

Fireplace needs to be reachable by real users at a cost proportional to its current audience,
which is approximately zero. A single VPS at roughly $20-40/month running the compose stack,
with Caddy in front terminating TLS, is the appropriate posture — and stays appropriate until
something in the system develops a scaling need the box cannot absorb.

Establishing that revealed a gap between how Fireplace is assumed to run and how it actually
runs. **There are no Dockerfiles in the repository, and `docker-compose.yml` contains no
application services.** Compose is infrastructure only: Consul, RabbitMQ, Redis, ten Postgres
containers, and (until ADR-0011) Temporal. Every Go service runs on the host under `air`.

So "run the compose stack on a VPS" is not a port-binding pass over an existing file. It is the
**first containerization of the application tier**, and it is the largest single piece of work in
the consolidation. Naming it here rather than discovering it mid-sequence.

A correctness problem falls out of the same gap. Every service binds and registers itself on
`localhost`:

```go
registry.Register(ctx, instanceID, serviceName, "localhost:"+grpcAddr)
listener, err := net.Listen("tcp", "localhost:"+grpcAddr)
```

That is correct for processes sharing a host under `air` and **fatal in containers**: a process
binding `localhost` inside a container accepts connections from nothing but itself. This has
never failed because nothing has ever been containerized.

Service discovery is the other thing the move changes. All six services register with Consul.
After ADR-0009 there are three, with static names, on one Docker network — which is precisely
the situation Docker's own DNS handles, and Consul becomes a container and a dependency earning
nothing. It is also cheap to remove, because the codebase already did the hard part:
`discovery.Registry` is an interface (`common/discovery/discovery.go`) and every call site goes
through `discovery.ServiceConnection(ctx, serviceName, registry)`
(`common/discovery/grpc.go:21`). A static implementation drops in behind it without touching a
single caller, and without disturbing the cached-`ClientConn` behaviour in the gateway's gRPC
clients.

Recorded without adversarial review — locked collaboratively during the consolidation scoping
session and not run through `challenge-me`.

## Decision

**One VPS, one compose stack, Caddy terminating TLS, and the application tier containerized for
the first time.**

1. **Containerize all three services.** A Dockerfile each for api-gateway, plan-service and
   insights-service, and compose entries to run them. This is new work, not a config edit.
2. **Fix the bind and registration addresses as a precondition.** `net.Listen("tcp",
   "localhost:"+grpcAddr)` becomes `net.Listen("tcp", ":"+grpcAddr)`, and services register
   under their container name rather than `localhost`. Nothing on the internal network can talk
   to anything until this lands.
3. **Only the gateway is published to the host.** Compose currently exposes every service and
   every database (`5301`-`5306`, `5556`, `6301`-`6306`, plus `5683`/`15683` for RabbitMQ,
   `6380` for Redis and `8520` for Consul). In production the gateway is the sole published
   port; databases, RabbitMQ, Redis and all gRPC ports (`7101`-`7106`) are reachable only on the
   internal Docker network. Local development may continue to publish whatever is convenient —
   this is a production posture, not a compose-file rewrite.
4. **Consul is removed.** A static, environment-backed `discovery.Registry` implementation
   replaces `consul.NewRegistry` in each service's `main.go`, resolving services by their compose
   name. Call sites are untouched because they already depend on the interface, and the gateway's
   cached `*grpc.ClientConn` and lazy-redial behaviour are unaffected. The Consul container and
   its volume leave the stack.
5. **Secrets leave the repository for production.** Compose environment variables remain
   acceptable locally. API keys and the six database credentials introduced by ADR-0010 are
   supplied to the production stack from outside version control.
6. **Backups are blocking before real users.** `pg_dump` on a cron to object storage, covering
   all three databases, **with a restore that has actually been performed at least once**. An
   untested backup is not a backup. This gates the first real user, not the first deploy.
7. **The trigger for leaving this posture is written down in advance:** insights-service needing
   to scale independently under LLM load. That is the expected first mover and the reason it
   stayed a service in ADR-0009. Secondary triggers: the shared Postgres instance becoming a
   contention point (ADR-0010), or one box's failure domain becoming unacceptable to users who
   now exist.

## Consequences

**Good**

- Total infrastructure cost is one VPS, against a system that until now had no deployment story
  at all.
- Containerization is a prerequisite for every future hosting option, so the work is not wasted
  if the box is outgrown — it is the thing that makes moving possible.
- Attack surface is one published port instead of nineteen.
- Removing Consul drops a container, a dependency and the registry wiring, and is reversible: if
  the platform ever runs on more than one host, swapping the `Registry` implementation back is
  the whole change.
- The move trigger exists before the pressure does, so the decision to leave gets made on the
  stated condition rather than during an incident.

**Bad / accepted**

- **Single point of failure, stated plainly.** One box hosts all three services, the Postgres
  instance, RabbitMQ and Redis. Its failure is total, and combined with ADR-0010's shared
  instance there is no partial-availability story at all. Correct for the current audience,
  and the first thing to revisit when that changes.
- **Containerization is substantial unplanned work** — three Dockerfiles, three compose entries,
  the bind fix, Caddy configuration, and a build path that does not exist today. It belongs at
  the end of the sequence, after the database consolidation and the service folding are verified
  locally under `air`.
- **Local development and production diverge.** `air` on the host stays the development loop
  while production runs containers, so a class of bug — path assumptions, the relative
  `file://migrations` source noted in ADR-0010, network addressing — can only appear in
  production. Mitigated by being able to run the compose stack locally, not eliminated.
- **No CI/CD is specified here.** Deployment mechanics are deliberately left open; this ADR fixes
  the topology, not the pipeline.
- **Secrets management names no tool.** "Out of the repository" is the constraint. The mechanism
  is an implementation choice, and one that should not be over-built for a single box.
