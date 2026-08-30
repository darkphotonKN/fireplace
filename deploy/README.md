# Deploying Fireplace

Single VPS, compose stack, Caddy terminating TLS (ADR-0012).

## The three compose files

| File | Loaded when | Contains |
|---|---|---|
| `docker-compose.yml` | always | every service; **publishes no host ports** |
| `docker-compose.override.yml` | automatically, local only | dev ports (8080, 5500, 6500, 5683, 15683, 6380) + the test Postgres |
| `docker-compose.prod.yml` | only when named explicitly | Caddy on 80/443, `restart: unless-stopped`, and every secret made **required** |

The base file holds the production posture. Compose loads `override.yml` automatically,
which is what makes local development convenient — and what makes production safe, because
production simply does not load it.

```bash
# local
docker compose up -d

# production
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

## Production checklist

1. **DNS first.** `FIREPLACE_DOMAIN` must already resolve to the box. Caddy obtains a
   Let's Encrypt certificate on boot and the ACME challenge fails otherwise.
2. **Secrets from outside the repo.** Copy `.env.example` to `.env` on the server, or export
   the variables in the unit/shell that runs compose. Every one is `${VAR:?}` in the prod
   overlay, so a missing value stops the deploy rather than starting with a default that is
   printed in this repository.
3. **Generate real passwords.** The dev defaults (`owner` / `app` / `fireplace`) exist so
   `docker compose up` works on a laptop. They must not survive to production.
4. **Database credentials are read twice.** The same values create the roles (via
   `scripts/db/init-databases.sh`) and are used by the containers. They only take effect on
   an **empty** volume — changing them later needs `docker compose down -v`, which destroys
   data. Set them correctly the first time.
5. **Firewall.** Only 80 and 443 need to be open. Nothing else publishes a port.
6. **Backups are not done yet** — that is I-0042, and ADR-0012 §6 makes it blocking before
   real users.

## What is reachable

Only Caddy. Postgres, RabbitMQ, Redis, plan-service and insights-service publish nothing and
are reachable solely on the internal Docker network; the gateway is reached only through
Caddy's reverse proxy.

```
  internet ──443──▶ caddy ──▶ api-gateway:8080
                                 │
                                 ├──gRPC──▶ plan-service:7103
                                 ├──gRPC──▶ insights-service:7106
                                 └──SQL───▶ postgres:5432 (three databases)
```

## Rollout order

Migrations run as one-shot `*-migrate` containers that must exit 0 before their service
starts (ADR-0010 §4). A failed migration stops that service from starting rather than
letting it serve against a half-applied schema, so a bad deploy fails closed.
