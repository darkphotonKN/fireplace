# ADR-0016 — Backups are droplet backups, not an in-stack backup container

Status: accepted
Date: 2026-10-20
Scope: root. Governs how Fireplace's data is backed up and restored in production.
Amends: **ADR-0012 §6** (backups: `pg_dump` on a cron to object storage, with a restore drill).
ADR-0012 stands in every other respect.
Related: ADR-0010 (the one Postgres instance being backed up), I-0042 (the backup container
this removes)

## Context

ADR-0012 §6 made backups blocking before real users: `pg_dump` of all three databases on a cron
to object storage, with a restore performed at least once. I-0042 built it as a `backup`
container. It ran `pg_dump` daily, uploaded the dumps through `rclone` to an S3-compatible
bucket, pruned old ones and shipped a `restore.sh` drill against the test instance.

Deploying for real changed the picture. Fireplace is going onto the owner's existing
DigitalOcean droplet, which already runs tomato-timer and after-dark behind an edge Caddy owned
by the tomato-timer stack. The deploy is meant to be as simple as possible.

- **The bucket existed only for the backup.** Nothing else in Fireplace uses S3. The backup
  container was the one reason production needed object-storage credentials, a provider choice
  and a bucket to manage.
- **The audience is one user.** The data is worth keeping. It is not worth an offsite pipeline
  with its own monitoring gap. Nothing alerted on a failed run anyway.
- **The droplet is shared, and the other apps have no backups.** A Fireplace-only dump does
  nothing for tomato-timer or after-dark. Droplet-level backups protect all three apps with one
  setting.

Recorded without adversarial review. The owner made this call during deploy planning, and it
was not run through `challenge-me`.

## Decision

**No in-stack backup container and no object storage.**

1. **Backups are DigitalOcean whole-droplet backups.** They are enabled on the droplet, outside
   the compose stack, and they cover every app on it.
2. **A Fireplace-only copy is an on-demand `pg_dumpall`**, taken by hand when wanted, before a
   risky migration for example:
   `ssh root@206.189.40.30 'docker exec fireplace_postgres pg_dumpall -U postgres' > fireplace-$(date +%F).sql`.
   Restoring it is `psql` from that file into a fresh stack. `deploy/README.md` holds the steps.
3. **The `backup` service, the `fireplace_backups` volume, `deploy/backup/` and every
   `BACKUP_*` / `RCLONE_S3_*` variable leave the repository.** Production needs no S3 keys.

## Consequences

**Good**

- The deploy loses a container, a volume, a bucket and five secrets. Nothing in production now
  depends on an object-storage provider.
- tomato-timer and after-dark get backups as a side effect, which they had none of.
- A restore is DigitalOcean's restore, not a script of ours that has to keep working.

**Bad / accepted**

- **Restores are whole-droplet.** Fireplace can't be rolled back on its own. Restoring the
  droplet rolls tomato-timer and after-dark back to the same point. The manual `pg_dumpall` is
  the only Fireplace-only path, and it exists only if someone took one.
- **RPO is the droplet backup frequency.** Whatever DigitalOcean's schedule is (daily or
  weekly, depending on the plan), that is the most data that can be lost. There is no
  point-in-time recovery.
- **No restore drill for Fireplace alone.** ADR-0012 §6 asked for a restore actually performed.
  That gate is gone. Nothing proves a droplet backup turns back into a working Fireplace until
  one is restored.
- **The backups are not offsite from the provider.** They live in DigitalOcean, the same
  provider as the droplet. Losing the account loses both.
- **Revisit when the audience grows.** More than one real user, or data whose loss would hurt
  someone other than the owner, is the trigger to bring back an offsite, per-app dump.
