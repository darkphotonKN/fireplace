# example-service — Specification

<!-- migrating to thin format: one line per capability, → FS-NNNN pointers -->

> Scope: decorative reference/template scaffold — the canonical shape new Fireplace services are cloned from. Platform maps: ../../CLAUDE.md.

## Status

Scaffold / reference. No product features. Its only job is to be a correct, copyable example of the standard service wiring.

## What it provides today

- A fully implemented `Ping` gRPC RPC (`"pong: <msg>"` + `served_by` hostname) for smoke-testing the stack.
- Declared AMQP topology: exchange `example.events` (topic, durable) + queue `example-service.events` (durable), with no bindings and a no-op consumer.
- Reference wiring for Consul registration, telemetry init, graceful shutdown, and handler→service→publisher/consumer layering.

## Intended future role

Remains a template. When a new domain service is extracted from the monolith, this directory is copied and adapted (see the "Cloning this for a new service" section in CLAUDE.md).

## gRPC Surface

`example.ExampleService`
- `Ping(PingRequest{message}) → PingResponse{reply, served_by}`

## Not in scope

- No database (no migrations; `DB_*` in `.env` are placeholders, unused).
- No real events — publisher is a logging stub, consumer has no bindings.
- No HTTP, no Redis, no calls to other services.
