# ADR-0017 — Deploy onto the shared droplet, behind its edge Caddy, from CI-built images

Status: accepted
Date: 2026-10-20
Scope: root. Governs where the production stack runs, what terminates TLS for it, and how its
images reach the box.
Amends: **ADR-0012 §3** (Caddy fronting the gateway) and fills the pipeline ADR-0012 left open.
ADR-0012 stands in every other respect.
Related: ADR-0016 (backups on the same droplet), tomato-timer's `deploy/README.md` (the edge
Caddy's owner), after-dark ADR-0005 (the first app to join that Caddy)

## Context

ADR-0012 put Fireplace on a single VPS with Caddy terminating TLS, and the repo realized that as
a `caddy` service inside Fireplace's own production compose file, binding 80 and 443.

The VPS turned out to be one that already exists: the owner's DigitalOcean droplet
(206.189.40.30, 2 vCPU, 4 GB), which runs tomato-timer and After Dark. tomato-timer's stack owns
a Caddy that holds 80/443 and issues Let's Encrypt certificates. After Dark already joined it:
its services sit on that stack's `tomato-timer_default` network, and its site block lives in the
shared Caddyfile. Only one process can bind 80/443, so a second Caddy is not an option, and
duplicating that setup was never the point.

Two other facts shaped the decision:

- **The droplet can't afford to compile.** tomato-timer and After Dark both build images in
  GitHub Actions and push to GHCR, because a Go build on a small shared box starves the apps
  already serving. Fireplace has three Go services, so this applies three times over.
- **The frontend goes to Vercel**, so the API is cross-origin and must be HTTPS (an HTTPS page
  cannot call plain HTTP). Fireplace has no domain of its own. `sslip.io` maps
  `fireplace.206-189-40-30.sslip.io` to the droplet's IP with no DNS setup, and After Dark used
  the same trick before it bought a domain.

Recorded without adversarial review. The owner approved the plan during deploy planning, and it
was not run through `challenge-me`.

## Decision

1. **No Caddy in Fireplace's stack.** The droplet's edge Caddy, owned by tomato-timer, terminates
   TLS for Fireplace too. `deploy/edge-site.caddy` holds the site block. It is appended to
   `/opt/tomato-timer/Caddyfile` and mirrored in tomato-timer's repo.
2. **Only the gateway joins the edge network.** `api-gateway` attaches to the external
   `tomato-timer_default` network and is proxied to by container name
   (`fireplace_api_gateway:8080`). Postgres, RabbitMQ, Redis, plan-service and insights-service
   stay on `fireplace-network` alone. Nothing publishes a host port, which keeps ADR-0012 §3's
   "one way in" intact.
3. **The API hostname is `fireplace.206-189-40-30.sslip.io`** until Fireplace has a domain.
4. **Images are built in CI, never on the droplet.** `.github/workflows/deploy.yml` builds the
   three service images, pushes them to GHCR as `:latest` and `:<sha>`, then SSHes in to sync the
   compose files, pull the `:<sha>` images and restart. The prod overlay resets the base file's
   `build:` sections, so the box can't fall back to compiling.
5. **CORS origins come from `CORS_ALLOWED_ORIGINS`.** The gateway allowed only localhost before.
   Production sets the Vercel origin. Unset still means the local dev servers.

## Consequences

**Good**

- One droplet and one bill for three apps. Fireplace adds no TLS, DNS or certificate work.
- Deploys pull prebuilt images and take seconds. Rolling back means pinning an older `:<sha>`.
- The deploy shape matches tomato-timer and After Dark, so one mental model covers all three.

**Bad / accepted**

- **Fireplace depends on another repo's stack.** If tomato-timer's Caddy is down, Fireplace is
  unreachable. The hostname lives in a Caddyfile that Fireplace's deploy doesn't sync, so
  changing it means editing the droplet plus a `caddy reload`.
- **Three apps now share the edge Caddy.** After Dark's ADR-0005 named this as the point where
  the edge Caddy should move into its own stack that every app joins. That move is deferred, not
  rejected.
- **`sslip.io` is a third-party resolver.** If it is down or rate-limits Let's Encrypt for its
  shared zone, the hostname stops resolving or the certificate stops renewing. A real domain is
  the fix, and it is a one-line change in the site block.
- **No test job gates the deploy yet.** The image build is the only compile check. Fireplace's
  build/test CI is tracked separately (issue #46).
- **Every app on the droplet shares 4 GB.** Fireplace adds roughly 0.5–0.7 GB, so the stack's
  memory has to be watched, not assumed.
