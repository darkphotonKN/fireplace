#!/bin/sh
# Restore one database from object storage.
#
#   restore.sh <database> [stamp]
#
# With no stamp, restores the most recent dump. DEFAULTS TO THE TEST INSTANCE
# (RESTORE_HOST=postgres-test) so the drill can be run any day without risking
# production; point RESTORE_HOST at the live instance only deliberately.
#
# The drill this exists for is in deploy/README.md. ADR-0012 §6 treats an
# untested backup as no backup.

set -eu

DB="${1:?usage: restore.sh <database> [stamp]}"
STAMP="${2:-}"

: "${BACKUP_REMOTE:?BACKUP_REMOTE must be set}"
: "${RESTORE_HOST:=postgres-test}"
: "${RESTORE_PORT:=5432}"
: "${RESTORE_SUFFIX:=_test}"

case "$DB" in
  fireplace_gateway)  OWNER=fireplace_gateway_owner;  PW="${GATEWAY_OWNER_PASSWORD:?}"  ;;
  fireplace_plans)    OWNER=fireplace_plans_owner;    PW="${PLANS_OWNER_PASSWORD:?}"    ;;
  fireplace_insights) OWNER=fireplace_insights_owner; PW="${INSIGHTS_OWNER_PASSWORD:?}" ;;
  *) echo "unknown database: $DB" >&2; exit 1 ;;
esac

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

if [ -z "$STAMP" ]; then
  echo "→ finding most recent dump of ${DB}"
  FILE=$(rclone lsf "${BACKUP_REMOTE}/${DB}/" | sort | tail -1)
  [ -n "$FILE" ] || { echo "✗ no dumps found for ${DB}" >&2; exit 1; }
else
  FILE="${DB}_${STAMP}.sql.gz"
fi

echo "→ fetching ${FILE}"
rclone copyto "${BACKUP_REMOTE}/${DB}/${FILE}" "$WORK/$FILE"

TARGET="${DB}${RESTORE_SUFFIX}"
echo "→ restoring into ${TARGET} on ${RESTORE_HOST}"

# --clean --if-exists in the dump drops each object before recreating it, so a
# repeat drill against a populated target is idempotent rather than a pile of
# duplicate-key errors.
gunzip -c "$WORK/$FILE" \
  | PGPASSWORD="$PW" psql --host="$RESTORE_HOST" --port="$RESTORE_PORT" \
      --username="$OWNER" --dbname="$TARGET" --quiet \
      -v ON_ERROR_STOP=1

echo "✓ restored ${DB} → ${TARGET}"
# Verify with REAL counts, not pg_stat_user_tables.n_live_tup: the statistics
# collector has not run yet immediately after a restore, so n_live_tup reads 0
# across the board and the check reassures without proving anything.
echo "→ verifying: real row counts per table"
PGPASSWORD="$PW" psql --host="$RESTORE_HOST" --port="$RESTORE_PORT" \
  --username="$OWNER" --dbname="$TARGET" -tAc "
    SELECT string_agg(fmt, E'\n' ORDER BY fmt) FROM (
      SELECT '  ' || c.relname || ': ' ||
             (xpath('/row/c/text()',
               query_to_xml('SELECT count(*) AS c FROM public.' || quote_ident(c.relname),
                            false, true, '')))[1]::text || ' rows' AS fmt
      FROM pg_class c
      JOIN pg_namespace n ON n.oid = c.relnamespace
      WHERE n.nspname = 'public' AND c.relkind = 'r'
    ) s;"

TABLES=$(PGPASSWORD="$PW" psql --host="$RESTORE_HOST" --port="$RESTORE_PORT" \
  --username="$OWNER" --dbname="$TARGET" -tAc \
  "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';")
if [ "$(echo "$TABLES" | tr -d ' ')" = "0" ]; then
  echo "✗ restore produced NO tables — the dump was empty or did not apply" >&2
  exit 1
fi
echo "✓ drill complete: ${TARGET} has ${TABLES} tables"
