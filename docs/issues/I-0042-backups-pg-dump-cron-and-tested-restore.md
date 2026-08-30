---
id: I-0042
status: open
implements: ADR-0012
blocked_by: [I-0041]
labels: [enhancement]
title: "ADR-0012: pg_dump backups to object storage with a restore that has actually been run"
---
Implements ADR-0012 §6

## What to Build

Scheduled `pg_dump` of all three databases to object storage, and a restore that has been
performed rather than merely written down.

- Cron on the VPS dumping `fireplace_gateway`, `fireplace_plans` and `fireplace_insights`
- Upload to object storage, with credentials supplied per I-0041 (nothing in the repo)
- A documented retention window
- **A restore actually performed at least once**, into a scratch environment, verified by
  querying real rows out of the restored databases

**This issue gates the first real user, not the first deploy.** ADR-0012 §6 states it as
blocking, and the reason is that an untested backup is not a backup — it is an assumption with a
cron entry. The restore drill is the deliverable here; the dump script is the easy half.

Worth noting for whoever picks this up: the same `pg_dump` per database is the extraction path in
ADR-0010 §6. Getting it working here means the "move insights-service to its own instance" story
is already exercised.

## Acceptance Criteria

- [ ] Cron dumps all three databases on a stated schedule
- [ ] Dumps land in object storage and are verified present, not merely reported as uploaded
- [ ] Credentials are supplied from outside version control
- [ ] Retention window is documented
- [ ] **A restore has been performed into a scratch environment and real rows queried from it** —
      recorded in the PR description with what was restored and what was checked
- [ ] Restore procedure is written down well enough to follow under pressure

## Blocked By

I-0041 — backups run against the deployed production stack, and the object-storage credentials
arrive with the secrets work.

## Spec Reference

ADR-0012 §6 (backups blocking before real users, with a restore that has actually been performed).
Related: ADR-0010 §6, which uses the same per-database `pg_dump` as the extraction path.
