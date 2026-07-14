# orchestrator-service — Specification

<!-- migrating to thin format: one line per capability, → FS-NNNN pointers -->

> Scope: cross-service coordination scaffold — infrastructure for future multi-service (saga-style) workflows over gRPC + AMQP. Platform maps: ../../CLAUDE.md.

## Status

Scaffold / reference. No product features yet. A working `Ping` RPC and declared AMQP topology exist; real orchestration is stubbed/TODO.

## What it provides today

- A fully implemented `Ping` gRPC RPC (`"pong: <msg>"` + `served_by` hostname).
- Declared AMQP topology: exchange `orchestrator.events` (topic, durable) + queue `orchestrator-service.events` (durable), with no bindings and a no-op consumer.
- Wiring that injects the AMQP publisher into the service and the Consul registry into setup, ready for future publishing and downstream gRPC discovery.

## Intended future role

Coordinate multi-service workflows (saga-style): consume domain events from other services' exchanges, call downstream services over gRPC (discovered via Consul), and publish orchestration events — all without owning any persistent domain data.

## gRPC Surface

`orchestrator.OrchestratorService`
- `Ping(PingRequest{message}) → PingResponse{reply, served_by}`

## Not in scope

- No database — **by design** (no `DB_*` config, no postgres in compose). State, if ever needed, would live elsewhere.
- No real events yet — publisher is a logging stub; consumer has no bindings.
- No HTTP, no Redis. Downstream gRPC clients not yet wired.
