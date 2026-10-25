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

**Precondition: the target's `fireplace_insights` already has the `vector` extension.**
`rag_chunks` is typed `vector(512)` (ADR-0018), so a target without it loads every other table
and silently drops that one. A fresh volume on the compose image gets the extension from
`init-databases.sh`; anything else needs the one-off below first. The restore checks and
refuses:

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d postgres
if docker exec fireplace_postgres psql -U postgres -d fireplace_insights -tAc \
     "SELECT 1 FROM pg_extension WHERE extname = 'vector'" | grep -q 1; then
  docker exec -i fireplace_postgres psql -U postgres -d postgres < fireplace-YYYY-MM-DD.sql
  docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
else
  echo "refusing to restore: fireplace_insights has no vector extension (see 'The vector extension on an existing volume')" >&2
fi
```

`init-databases.sh` has already created the roles and the three empty databases on first boot,
so expect "already exists" errors for those. The tables and rows still load. The dump's own
`CREATE EXTENSION IF NOT EXISTS vector` is a no-op on a target that passed the check. The dump
and the restore both run as the superuser, so the superuser-owned extension replays cleanly.
A plain `postgres:17` instance can never be a restore target for `fireplace_insights`.

## The vector extension on an existing volume

`rag_chunks` (insights-service) needs pgvector's `vector` type in `fireplace_insights`
(ADR-0018, FS-DF4ZB R31c). pgvector is not a trusted extension, so only the instance
superuser can create it: never a migration, never an app role. `init-databases.sh` does it on
an **empty** volume. Every volume that already exists needs it done once by hand, **before**
the `rag_chunks` migration reaches it. Skip this and the migration fails with
`type "vector" does not exist`.

The order is always: the pgvector image, then the extension, then the migration. The image is
`pgvector/pgvector:0.8.7-pg17-trixie`: the same Postgres major and the same Debian release
(glibc 2.41) as the `postgres:17` image the volumes were created on, so the data directory
carries over without a collation-version mismatch. `docker exec ... psql -U postgres` connects
over the container's local socket, so no superuser password is typed. Each command is
idempotent.

**Production** (the owner runs this, on the droplet, **before** the branch that adds
`rag_chunks` is merged to `main`):

Do not leave any of this to the deploy. Every push to `main` deploys, and the deploy is one
`docker compose up -d`: it recreates Postgres on the pgvector image and runs `insights-migrate`
in the same breath, with no step in between for the extension. The migration then fails,
`schema_migrations` is left dirty at version 3, and insights-service does not start. So the
extension goes in by hand first, and the merge comes after.

```bash
# Pre-check: the current container's glibc. Must be 2.41 (Debian trixie) for the -trixie image.
# If it says 2.36, STOP: the volume was created on bookworm and a trixie image would raise a
# collation-version mismatch. Pick the matching image tag before going on.
ssh root@206.189.40.30 'docker exec fireplace_postgres ldd --version | head -1'
# 0. Take a dump first (Backups, above).
# 1. Move Postgres to the pgvector image: copy the branch's docker-compose.yml over (from a
#    checkout of the branch), then recreate only Postgres. A short database outage for the whole
#    stack; no other container is touched and no migration runs.
scp docker-compose.yml root@206.189.40.30:/opt/fireplace/
ssh root@206.189.40.30 'cd /opt/fireplace && docker compose pull postgres && docker compose up -d postgres'
# 2. The extension, as the instance superuser, in fireplace_insights only.
ssh root@206.189.40.30 'docker exec fireplace_postgres psql -v ON_ERROR_STOP=1 -U postgres -d fireplace_insights -c "CREATE EXTENSION IF NOT EXISTS vector;"'
# Pre-merge check: vector is listed. Do not merge until it is.
ssh root@206.189.40.30 'docker exec fireplace_postgres psql -U postgres -d fireplace_insights -c "\dx vector"'
# 3. Merge the branch. The deploy it triggers syncs the same docker-compose.yml and runs
#    insights-migrate, which now finds the type.
```

If another deploy from `main` lands between step 1 and the merge, it copies `main`'s
`docker-compose.yml` back and recreates Postgres on `postgres:17`. That is harmless: no table
uses `vector` yet, the extension stays in the catalog, and the merge's deploy moves the
container back to the pgvector image before the migration runs. Rerun the pre-merge check
before merging anyway.

**Local dev:**

```bash
docker compose up -d postgres    # recreates the container on the new image; the volume stays
docker exec fireplace_postgres psql -v ON_ERROR_STOP=1 -U postgres -d fireplace_insights \
  -c 'CREATE EXTENSION IF NOT EXISTS vector;'
make -C services/insights-service migrate-up
```

**Test** (`postgres-test`, port 6500):

```bash
docker compose up -d postgres-test
docker exec fireplace_postgres_test psql -v ON_ERROR_STOP=1 -U postgres -d fireplace_insights_test \
  -c 'CREATE EXTENSION IF NOT EXISTS vector;'
make -C services/insights-service migrate-up DB_PORT=6500 DB_NAME=fireplace_insights_test
```

Test data is disposable, so dropping the test volume also works: the next boot runs
`init-databases.sh`, which creates the extension.

Check it landed in `fireplace_insights` and nowhere else:

```bash
docker exec fireplace_postgres psql -U postgres -tAc "SELECT datname FROM pg_database WHERE datname LIKE 'fireplace%'" |
  while read -r db; do
    echo "$db: $(docker exec fireplace_postgres psql -U postgres -d "$db" -tAc "SELECT count(*) FROM pg_extension WHERE extname = 'vector'")"
  done
```

**If the migration ran first.** It fails as one transaction, so none of it is applied, but
golang-migrate records the schema as dirty at the `rag_chunks` version and refuses to go on.
Create the extension (step 2), set the version back to the one before `rag_chunks`
(`000003_create_rag_chunks_table` at the time of writing, so 2), then migrate again:

```bash
docker exec fireplace_postgres psql -v ON_ERROR_STOP=1 -U postgres -d fireplace_insights \
  -c "UPDATE schema_migrations SET version = 2, dirty = false WHERE version = 3 AND dirty;"
```

Locally, `make -C services/insights-service migrate-fix version=2` does the same.

A future pgvector upgrade (`ALTER EXTENSION vector UPDATE`) is the same kind of superuser
one-off: bump the image tag, then run it in `fireplace_insights`.
