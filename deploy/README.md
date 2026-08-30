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
6. **Configure and drill backups before the first real user** (ADR-0012 §6). See below.

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

## Backups and the restore drill

A `backup` container runs `pg_dump` over all three databases daily at 03:00 (container
timezone), gzips each, uploads to object storage via `rclone`, and prunes anything older
than `BACKUP_RETENTION_DAYS` (default 14).

It only prunes **after every dump succeeded** — pruning on a failed run would delete good old
backups on exactly the day the new ones are bad. A dump under 1 KB is refused rather than
uploaded, because a truncated dump is the failure that looks like success.

`rclone` speaks S3, Cloudflare R2, Backblaze B2 and DigitalOcean Spaces through the same
config; see `.env.example` for the per-provider variables. **Put the bucket on a different
machine than the VPS** — a backup on the box you are protecting against losing is not a
backup. Make the bucket private and enable its server-side encryption; the dumps contain
user emails and password hashes.

### Running it

```bash
# on demand, any environment
docker compose --profile backup run --rm backup backup.sh

# restore the latest dump into the TEST instance (safe, this is the drill)
docker compose --profile backup run --rm backup restore.sh fireplace_plans

# restore a specific dump
docker compose --profile backup run --rm backup restore.sh fireplace_plans 20260830T174511Z
```

In production the container runs as a daemon on the schedule; the profile only keeps it out
of ordinary local development.

### The drill

ADR-0012 §6 treats an untested backup as no backup, so `restore.sh` targets
`postgres-test` by default and can be run any day without touching production. It restores
into `<database>_test`, prints **real row counts per table**, and fails if the restore
produced no tables.

Run it after any change to the database layer, and on a schedule you actually keep. What it
proves is not that a file exists in a bucket but that the file turns back into a database.

**Restoring into production** means setting `RESTORE_HOST=postgres` and `RESTORE_SUFFIX=`
deliberately. Stop the services first — the dumps carry `--clean --if-exists`, so they drop
and recreate every object as they apply.

### Known limits

- **Nothing alerts you.** A failed backup exits non-zero and says so in `docker logs`, but no
  one is paged. Wire the container's exit status into whatever monitoring exists before
  relying on it.
- **Daily granularity.** Worst case you lose 24 hours. Point-in-time recovery would need WAL
  archiving, which is a bigger decision than this ADR makes.
- **The dumps are not encrypted by this tooling.** Bucket-level encryption plus a private
  bucket is the intended posture; an `rclone crypt` remote is the next step if the data
  warrants it, at the cost of a key you must not lose.
