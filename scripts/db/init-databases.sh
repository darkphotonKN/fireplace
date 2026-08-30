#!/bin/bash
# ADR-0010: three databases on one Postgres instance, two roles each.
#
# Runs ONCE, on first boot of an empty data volume (Postgres runs everything in
# /docker-entrypoint-initdb.d/ then never again). To re-run it after changing
# this file you must drop the volume:  docker compose down -v
#
# Isolation here is enforced by the engine, not by convention: PostgreSQL has no
# cross-database query, so <name>_app cannot reach another service's data even
# with a valid connection. The role split is the second layer — the app role
# holds DML only, so a compromised or injected process cannot DROP TABLE.
#
# DB_SUFFIX lets the same script build the test tier (ADR-0010 §5), which
# mirrors production exactly rather than routing around the grant model.

set -euo pipefail

SUFFIX="${DB_SUFFIX:-}"

# Passwords come from the environment. Compose supplies them locally; production
# supplies them from outside the repo (ADR-0012 §5). Defaults exist so a local
# `docker compose up` works out of the box and are not fit for anything else.
create_db() {
  local name="$1" owner_pw="$2" app_pw="$3"
  local db="${name}${SUFFIX}"
  local owner="${name}_owner"
  local app="${name}_app"

  echo "→ ${db}: creating roles ${owner} (DDL) and ${app} (DML only)"

  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<-SQL
	DO \$\$ BEGIN
	  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '${owner}') THEN
	    CREATE ROLE ${owner} LOGIN PASSWORD '${owner_pw}';
	  END IF;
	  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '${app}') THEN
	    CREATE ROLE ${app} LOGIN PASSWORD '${app_pw}';
	  END IF;
	END \$\$;
SQL

  # Owned by the migration role, so every object migrations create belongs to it.
  # CREATE DATABASE cannot run inside a DO block or a transaction, and psql's
  # \gexec is not interpreted by -c, so the existence check is done in shell.
  if ! psql -tAc "SELECT 1 FROM pg_database WHERE datname = '${db}'" \
       --username "$POSTGRES_USER" --dbname postgres | grep -q 1; then
    psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres \
      -c "CREATE DATABASE ${db} OWNER ${owner}"
  fi

  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "${db}" <<-SQL
	-- Extensions are created here, as superuser, so migrations never depend on
	-- the owner's right to install them.
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
	CREATE EXTENSION IF NOT EXISTS "pg_trgm";

	-- Nobody but this database's own roles gets in. PUBLIC can connect to any
	-- database by default, which would undo the point of separate roles.
	REVOKE ALL ON DATABASE ${db} FROM PUBLIC;
	GRANT CONNECT ON DATABASE ${db} TO ${owner}, ${app};

	-- PUBLIC can CREATE in the public schema by default too. The app role must
	-- not be able to create anything, so that is revoked and only the owner is
	-- granted it back.
	REVOKE ALL ON SCHEMA public FROM PUBLIC;
	GRANT ALL ON SCHEMA public TO ${owner};
	GRANT USAGE ON SCHEMA public TO ${app};

	-- Existing objects (none on a fresh volume, but this script is re-runnable).
	GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO ${app};
	GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO ${app};

	-- The load-bearing grant: migrations run LATER, as \${owner}, and create
	-- tables this script never sees. Without a default privilege the app role
	-- would be unable to read its own service's tables and the failure would
	-- appear at first query, not here.
	ALTER DEFAULT PRIVILEGES FOR ROLE ${owner} IN SCHEMA public
	  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO ${app};
	ALTER DEFAULT PRIVILEGES FOR ROLE ${owner} IN SCHEMA public
	  GRANT USAGE, SELECT ON SEQUENCES TO ${app};
SQL
}

create_db fireplace_gateway  "${GATEWAY_OWNER_PASSWORD:-owner}"  "${GATEWAY_APP_PASSWORD:-app}"
create_db fireplace_plans    "${PLANS_OWNER_PASSWORD:-owner}"    "${PLANS_APP_PASSWORD:-app}"
create_db fireplace_insights "${INSIGHTS_OWNER_PASSWORD:-owner}" "${INSIGHTS_APP_PASSWORD:-app}"

echo "✓ three databases, six roles ready${SUFFIX:+ (suffix: ${SUFFIX})}"
