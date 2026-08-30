---
id: I-0041
status: done
implements: ADR-0012
blocked_by: [I-0040]
labels: [enhancement]
title: "ADR-0012: production posture — Caddy TLS, gateway-only port exposure, secrets out of the repo"
---
Implements ADR-0012 §3, §5

## What to Build

Take the containerized stack from I-0040 and give it a production posture on the VPS.

**1. Caddy in front, terminating TLS.** Reverse-proxying to the gateway only.

**2. Publish the gateway and nothing else.** Compose currently exposes every service and every
database on the host: `5301`-`5306`, `5556`, `6301`-`6306`, plus `5683`/`15683` for RabbitMQ,
`6380` for Redis and `8520` for Consul (the last of which is gone as of I-0040). In production the
gateway is the only published port; Postgres, RabbitMQ, Redis and every gRPC port (`7103`, `7106`)
are reachable only on the internal Docker network.

This is a **production posture, not a compose-file rewrite** — local development may keep
publishing whatever is convenient. Use a compose override or an equivalent split rather than
making the local loop harder to work with.

**3. Secrets out of version control.** The six database credentials introduced by I-0039 and the
OpenAI API key are supplied to the production stack from outside the repository. No tool is
mandated by ADR-0012 and none should be over-built for a single box — environment files delivered
out of band are sufficient, provided nothing lands in git.

## Acceptance Criteria

- [ ] Caddy terminates TLS and reverse-proxies to the gateway
- [ ] `docker compose ps` on the VPS shows exactly one published port
- [ ] Postgres, RabbitMQ, Redis and all gRPC ports are unreachable from outside the host
- [ ] The local development loop still works without hand-editing the production compose file
- [ ] No database credential or API key is present anywhere in version control
- [ ] The gateway is reachable over HTTPS with a valid certificate
- [ ] The plan ↔ insights event flow works end to end on the deployed stack

## Blocked By

I-0040 — there is nothing to expose or proxy to until the application tier runs in containers.

## Spec Reference

ADR-0012 §3 (only the gateway is published to the host), §5 (secrets leave the repository for
production).
