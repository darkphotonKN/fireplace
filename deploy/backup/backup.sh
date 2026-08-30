#!/bin/sh
# Dump all three Fireplace databases and upload them to object storage.
#
# ADR-0012 §6. The dumps are the easy half — a backup nobody has restored is an
# assumption with a cron entry, so see restore.sh and the drill in
# deploy/README.md.
#
# Exits NON-ZERO if any database fails, so a failure is visible in `docker logs`
# and to any monitoring watching the container, rather than being swallowed by
# the loop and leaving a gap nobody notices until a restore is needed.

set -eu

: "${PGHOST:=postgres}"
: "${PGPORT:=5432}"
: "${BACKUP_REMOTE:?BACKUP_REMOTE must be set (e.g. s3:fireplace-backups)}"
: "${RETENTION_DAYS:=14}"

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

# Dumps run as each database's OWNER (ADR-0010 §2): the app roles hold DML only
# and cannot read schema definitions, so a dump taken as the app role restores
# to an empty database.
dump_one() {
  db="$1"; user="$2"; pw="$3"
  out="$WORK/${db}_${STAMP}.sql.gz"

  echo "→ dumping ${db}"
  # --no-owner / --no-privileges: the target instance creates its own roles via
  # init-databases.sh, so baking role names into the dump makes it restorable
  # ONLY where those exact roles already exist. This is also what makes the
  # ADR-0010 §6 extraction path work.
  PGPASSWORD="$pw" pg_dump \
      --host="$PGHOST" --port="$PGPORT" --username="$user" --dbname="$db" \
      --no-owner --no-privileges --clean --if-exists \
    | gzip -9 > "$out"

  # pg_dump's exit status is hidden by the pipe, so verify the artifact instead.
  # A truncated or empty dump is the failure mode that looks like success.
  size=$(wc -c < "$out")
  if [ "$size" -lt 1000 ]; then
    echo "✗ ${db}: dump is only ${size} bytes — refusing to upload" >&2
    return 1
  fi

  echo "→ uploading ${db} (${size} bytes)"
  rclone copyto "$out" "${BACKUP_REMOTE}/${db}/${db}_${STAMP}.sql.gz"
  echo "✓ ${db}"
}

failed=0
dump_one fireplace_gateway  fireplace_gateway_owner  "${GATEWAY_OWNER_PASSWORD:?}"  || failed=1
dump_one fireplace_plans    fireplace_plans_owner    "${PLANS_OWNER_PASSWORD:?}"    || failed=1
dump_one fireplace_insights fireplace_insights_owner "${INSIGHTS_OWNER_PASSWORD:?}" || failed=1

if [ "$failed" -ne 0 ]; then
  echo "✗ backup FAILED at ${STAMP} — not pruning, so nothing old is lost" >&2
  exit 1
fi

# Prune only after every dump succeeded. Pruning on a failed run would delete
# good old backups on the day the new ones are bad.
echo "→ pruning backups older than ${RETENTION_DAYS}d"
rclone delete --min-age "${RETENTION_DAYS}d" "$BACKUP_REMOTE" || true

echo "✓ backup complete: ${STAMP}"
