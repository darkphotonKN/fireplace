# Deploying Fireplace

The backend runs on the shared DigitalOcean droplet (`206.189.40.30`, also home to tomato-timer
and After Dark), behind that droplet's edge Caddy (ADR-0012 as amended by ADR-0017). The
frontend runs on Vercel.

- **API:** `https://fireplace.206-189-40-30.sslip.io`
- **Stack on the droplet:** `/opt/fireplace`
- **Images:** `ghcr.io/darkphotonkn/fireplace-{api-gateway,plan-service,insights-service}`

## Shape

```
push to main (backend paths)
  └─ GitHub Actions  .github/workflows/deploy.yml
       ├─ build   3 images ─push─▶ GHCR (:latest, :<sha>)
       └─ deploy  ssh droplet: sync compose files, pull :<sha>, up -d, verify HTTPS

  browser ──▶ Vercel (client/)
     │
     └──443──▶ edge Caddy (tomato-timer stack, holds 80/443 for every app)
                  │  tomato-timer_default network
                  ▼
               api-gateway:8080 ─┬─gRPC──▶ plan-service:7103
                                 ├─gRPC──▶ insights-service:7106
                                 └─SQL───▶ postgres:5432 (three databases)
                                 (+ RabbitMQ, Redis — fireplace-network only)
```

Only the gateway is on the edge network. Nothing in this stack publishes a host port.

## The three compose files

| File | Loaded when | Contains |
|---|---|---|
| `docker-compose.yml` | always | every service; **publishes no host ports** |
| `docker-compose.override.yml` | automatically, local only | dev ports (8080, 5500, 6500, 5683, 15683, 6380) + the test Postgres |
| `docker-compose.prod.yml` | only when named explicitly | GHCR images (no builds), the gateway on the edge network, `restart: unless-stopped`, and every secret made **required** |

The base file holds the production posture. Compose loads `override.yml` automatically,
which is what makes local development convenient — and what makes production safe, because
production simply does not load it.

```bash
# local
docker compose up -d

# production (on the droplet, /opt/fireplace/.env sets COMPOSE_FILE so plain
# `docker compose` picks up both files)
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

## First-time setup

1. **`/opt/fireplace/.env` on the droplet**, mode `600`. Start from `.env.example` and add
   `COMPOSE_FILE=docker-compose.yml:docker-compose.prod.yml`. Every value is `${VAR:?}` in the
   prod overlay, so a missing one stops the deploy instead of booting on a default printed in
   this repository. Generate passwords with `openssl rand -hex 24`. The dev defaults
   (`owner` / `app` / `fireplace`) must not survive to production.
2. **Database credentials are read twice.** The same values create the roles (via
   `scripts/db/init-databases.sh`) and are used by the containers. They only take effect on an
   **empty** volume, so changing them later needs `docker compose down -v`, which destroys
   data. Set them correctly the first time.
3. **The edge site block.** Append `deploy/edge-site.caddy` to `/opt/tomato-timer/Caddyfile`,
   mirror it in tomato-timer's `deploy/Caddyfile` (and regenerate its `bootstrap.sh`), then
   `docker exec tomato_timer_caddy caddy reload --config /etc/caddy/Caddyfile`. Reload checks
   the file first, so a bad edit is refused rather than taking the other sites down.
4. **Repo secrets** `DROPLET_HOST`, `DROPLET_USER`, `DROPLET_SSH_KEY`: the same values
   tomato-timer and after-dark use. Then run the `deploy` workflow (Actions → deploy → Run
   workflow).
5. **Vercel.** Import the repo with root directory `client/` and set
   `NEXT_PUBLIC_API_BASE_URL=https://fireplace.206-189-40-30.sslip.io`. Put the Vercel origin in
   `CORS_ALLOWED_ORIGINS` in `/opt/fireplace/.env` and `docker compose up -d api-gateway`.
6. **Turn on DigitalOcean droplet backups** (ADR-0016). See below.

## Day-to-day

Every push to `main` that touches the backend deploys. `client/`, `docs/` and Markdown changes
are skipped. Actions → deploy → Run workflow redeploys without a commit.

```bash
ssh root@206.189.40.30 'cd /opt/fireplace && docker compose ps'
ssh root@206.189.40.30 'cd /opt/fireplace && docker compose logs -f api-gateway'

# roll back: pin an older commit's images in .env, then
ssh root@206.189.40.30 'cd /opt/fireplace && docker compose up -d'
```

## Rollout order

Migrations run as one-shot `*-migrate` containers that must exit 0 before their service
starts (ADR-0010 §4). A failed migration stops that service from starting rather than
letting it serve against a half-applied schema, so a bad deploy fails closed.

## Backups

Backups are DigitalOcean's whole-droplet backups (ADR-0016, amending ADR-0012 §6). They are
enabled on the droplet itself, not in this stack, and they cover the other apps sharing it.
There is no backup container and no object storage.

A restore is whole-droplet, so it rolls back every app on the box together. For a copy of
Fireplace's data alone, dump it by hand:

```bash
ssh root@206.189.40.30 'docker exec fireplace_postgres pg_dumpall -U postgres' > fireplace-$(date +%F).sql
```

To restore, start only Postgres on an empty `fireplace_pgdata` volume, feed it the file with
`psql`, then bring up the rest. Starting the services first would let the migrate containers
create tables the dump then collides with.

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d postgres
docker exec -i fireplace_postgres psql -U postgres -d postgres < fireplace-YYYY-MM-DD.sql
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

`init-databases.sh` has already created the roles and the three empty databases on first boot,
so expect "already exists" errors for those. The tables and rows still load.
