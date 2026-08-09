# Variables
MIGRATIONS_PATH = ./migrations

# Load the .env file if it exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Construct the DB_STRING dynamically
DB_STRING=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable
MIGRATE_COMMAND = migrate -path $(MIGRATIONS_PATH) -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable"

build:
	@go build -o bin/app ./cmd/

run: build
	@./bin/app

dev: 
	@air

# Run tests with verbose output and coverage
test:
	@go test -v ./... -cover

# Run tests with coverage output and preview in a browser
test-preview:
	@go test ./filename/ -coverprofile=coverage.out 
	@go tool cover -html=coverage.out

# Migration commands using golang-migrate
migrate-up:
	@migrate -path ./migrations -database "$(DB_STRING)" up

migrate-down:
	@migrate -path ./migrations -database "$(DB_STRING)" down

migrate-status:
	@migrate -path ./migrations -database "$(DB_STRING)" version

migrate-fix:
	@if [ -z "$(version)" ]; then \
		echo "Usage: make migrate-fix version=<version_number>"; \
		echo "Current status:"; \
		$(MIGRATE_COMMAND) version; \
		exit 1; \
	fi; \
	echo "Fixing dirty migration by setting version to $(version)..."; \
	$(MIGRATE_COMMAND) force $(version)


migrate-down-to:
	@if [ -z "$(VERSION)" ]; then \
		echo "Usage: make migrate-down-to VERSION=<version>"; \
		exit 1; \
	fi; \
	migrate -path ./migrations -database "$(DB_STRING)" down $(VERSION)

migrate-reset:
	@migrate -path ./migrations -database "$(DB_STRING)" down
	@migrate -path ./migrations -database "$(DB_STRING)" up

migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Usage: make migrate-create NAME=<migration_name>"; \
		exit 1; \
	fi; \
	migrate create -ext sql -dir ./migrations -seq $(NAME)

# Database management commands (USE WITH CAUTION!)
db-force-clean:
	@echo "⚠️  WARNING: This will reset your database to a clean state!"
	@echo "Current migration status:"
	@$(MIGRATE_COMMAND) version || true
	@echo ""
	@echo "Attempting to fix dirty state and reset..."
	@VERSION=$$($(MIGRATE_COMMAND) version 2>/dev/null | grep -o '[0-9]*' | head -1); \
	if [ ! -z "$$VERSION" ]; then \
		echo "Fixing dirty state at version $$VERSION..."; \
		$(MIGRATE_COMMAND) force $$VERSION; \
	fi
	@echo "Rolling back all migrations..."
	@$(MIGRATE_COMMAND) down -all || true
	@echo "Running all migrations fresh..."
	@$(MIGRATE_COMMAND) up
	@echo "✅ Database reset complete!"
	@$(MIGRATE_COMMAND) version

db-drop:
	@echo "⚠️  WARNING: This will DROP the database!"
	@psql -U $(DB_USER) -h $(DB_HOST) -p $(DB_PORT) -d postgres -c "DROP DATABASE IF EXISTS $(DB_NAME);"
	@echo "Database $(DB_NAME) dropped."

db-create:
	@echo "Creating database $(DB_NAME)..."
	@psql -U $(DB_USER) -h $(DB_HOST) -p $(DB_PORT) -d postgres -c "CREATE DATABASE $(DB_NAME);"
	@echo "Database $(DB_NAME) created."

db-drop-create: db-drop db-create
	@echo "Running migrations on fresh database..."
	@$(MIGRATE_COMMAND) up
	@echo "✅ Fresh database created and migrated!"
	@$(MIGRATE_COMMAND) version

# ---------------------------------------------------------------------------
# API docs — code-first OpenAPI 2.0 via swaggo/swag. See docs/api-conventions.md.
# The spec (docs/swagger.{json,yaml}) is a generated build artifact, never
# hand-edited. `make gen` == `go generate ./...`.
# ---------------------------------------------------------------------------

# Regenerate the OpenAPI 2.0 spec FROM the annotated handlers + edge DTOs.
gen:
	@go run github.com/swaggo/swag/cmd/swag init -g cmd/main.go -o docs --parseDependency --parseInternal

# Lint the generated spec: catches example-vs-schema drift + spec hygiene.
# Requires Node (npx). For a pure-Go setup, drop this and rely on `make gen`
# succeeding as the (weaker) net — tradeoff noted in docs/api-conventions.md.
lint:
	@npx --yes @stoplight/spectral-cli@6 lint docs/swagger.json

# Serve the generated spec in a local UI (standalone — no DB/Consul needed).
# The running gateway also serves it at http://localhost:8080/swagger/index.html
# (gin-swagger).
docs:
	@npx --yes @redocly/cli@latest preview-docs --port 8089 docs/swagger.json

# ---------------------------------------------------------------------------
# SERIALIZED contract (OpenAPI 3.x via Huma) — ADR-0002 plane 1.
#
# Distinct from the legacy `gen`/`lint`/`docs` targets above, which serve the
# swaggo OpenAPI 2.0 surface. Both surfaces coexist until the last endpoint
# serializes (ADR-0002 §8); these targets govern the new one.
#
# Command paths are declared in docs/agents/contract.md so generic skills can
# find them without hardcoding a toolchain.
# ---------------------------------------------------------------------------

OPENAPI_SPEC   = openapi.yaml
CLIENT_DIR     = ../../client/src/api/generated
SPECTRAL_RULES = ../../.spectral.yaml
OASDIFF_IGNORE = ../../.oasdiff-ignore

# oasdiff is PINNED and installed as a prebuilt binary rather than run via `go run`.
#
# Two reasons, both load-bearing:
#   1. `@latest` inside a gate is not reproducible — the gate can turn red on a day
#      nobody touched the contract.
#   2. EVERY oasdiff release requires go >= 1.26, while this module's directive is
#      1.25.0. `go run` bridges that silently by downloading a second toolchain
#      ("switching to go1.26.5"), which makes the gate depend on a network toolchain
#      fetch and breaks under GOTOOLCHAIN=off.
#
# A pinned binary keeps the GATE's toolchain independent of the SERVICE's — which is
# also what lets repos on different Go versions share this Makefile unchanged.
OASDIFF_VERSION = 1.28.0
TOOLS_DIR       = .tools
OASDIFF         = $(TOOLS_DIR)/oasdiff

# ONE shell block, not one line per step: make runs each recipe LINE in its own shell,
# so a multi-line guard would exit only its own line and let the rest proceed. That
# exact mistake produced two false greens here once already.
$(OASDIFF):
	@mkdir -p $(TOOLS_DIR)
	@set -e; \
	os=$$(uname -s | tr '[:upper:]' '[:lower:]'); \
	case "$$os" in \
	  darwin) asset="darwin_all" ;; \
	  linux) \
	    case "$$(uname -m)" in \
	      aarch64|arm64) asset="linux_arm64" ;; \
	      *)             asset="linux_amd64" ;; \
	    esac ;; \
	  *) echo "❌ unsupported OS '$$os' — install oasdiff v$(OASDIFF_VERSION) into $(TOOLS_DIR)/ by hand"; exit 1 ;; \
	esac; \
	echo "installing oasdiff v$(OASDIFF_VERSION) ($$asset)..."; \
	curl -sSL "https://github.com/oasdiff/oasdiff/releases/download/v$(OASDIFF_VERSION)/oasdiff_$(OASDIFF_VERSION)_$$asset.tar.gz" \
	  | tar -xz -C $(TOOLS_DIR) oasdiff
	@$(OASDIFF) --version

# Regenerate the OpenAPI 3.x spec FROM the typed Huma handler signatures.
# Types, not comments — this is what makes drift structurally impossible.
openapi:
	@if [ ! -d ./cmd/openapi ]; then \
		echo "SKIPPED: no ./cmd/openapi generator yet (expected until FS-0002 slice 3)"; \
	else \
		go run ./cmd/openapi > $(OPENAPI_SPEC) && echo "✅ wrote $(OPENAPI_SPEC)"; \
	fi

# Regenerate the TypeScript client FROM the committed spec. The generated client
# is the FE's ONLY import path for serialized endpoints (ADR-0002 §4).
client:
	@if [ ! -f $(OPENAPI_SPEC) ]; then \
		echo "SKIPPED: $(OPENAPI_SPEC) does not exist yet (expected until FS-0002 slice 3)"; \
	else \
		mkdir -p $(CLIENT_DIR) && \
		npx --yes openapi-typescript@7 $(OPENAPI_SPEC) -o $(CLIENT_DIR)/schema.d.ts && \
		echo "✅ wrote $(CLIENT_DIR)/schema.d.ts"; \
	fi

# Lint the SERIALIZED spec for contract depth (descriptions, examples, error schemas).
# The legacy `lint` target above still governs docs/swagger.json — do not merge them.
lint-contract:
	@if [ ! -f $(OPENAPI_SPEC) ]; then \
		echo "SKIPPED: $(OPENAPI_SPEC) does not exist yet (expected until FS-0002 slice 3)"; \
	else \
		npx --yes @stoplight/spectral-cli@6 lint --ruleset $(SPECTRAL_RULES) $(OPENAPI_SPEC); \
	fi

# Fail if the committed spec differs from the regenerated one. This is the gate
# that makes "committed openapi.yaml" trustworthy rather than aspirational.
openapi-diff:
	@if [ ! -f $(OPENAPI_SPEC) ]; then \
		echo "SKIPPED: $(OPENAPI_SPEC) does not exist yet (expected until FS-0002 slice 3)"; \
	else \
		cp $(OPENAPI_SPEC) /tmp/openapi.committed.yaml && \
		$(MAKE) --no-print-directory openapi && \
		{ diff -u /tmp/openapi.committed.yaml $(OPENAPI_SPEC) \
			|| { echo "❌ committed spec != regenerated — run 'make openapi' and commit"; exit 1; }; } && \
		echo "✅ committed spec matches regenerated"; \
	fi

# Breaking-change detection against the spec as committed on the base branch.
# Deliberate breaks pass ONLY via an entry in .oasdiff-ignore with a stated reason.
openapi-breaking: $(OASDIFF)
	@if [ ! -f $(OPENAPI_SPEC) ]; then \
		echo "SKIPPED: $(OPENAPI_SPEC) does not exist yet (expected until FS-0002 slice 3)"; \
	elif ! git show origin/main:services/api-gateway/$(OPENAPI_SPEC) > /tmp/openapi.base.yaml 2>/dev/null; then \
		rm -f /tmp/openapi.base.yaml; \
		echo "SKIPPED: no baseline spec on origin/main yet (first serialization)"; \
	else \
		$(OASDIFF) breaking /tmp/openapi.base.yaml $(OPENAPI_SPEC) \
			--err-ignore $(OASDIFF_IGNORE) --fail-on ERR; \
	fi

# Every plane-1 gate, in the order CI runs them.
gates: openapi-diff lint-contract openapi-breaking
	@echo "✅ contract gates complete"

.PHONY: run test migrate-up migrate-down migrate-status migrate-down-to migrate-reset migrate-create migrate-fix db-force-clean db-drop db-create db-drop-create gen lint docs openapi client lint-contract openapi-diff openapi-breaking gates








