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

.PHONY: run test migrate-up migrate-down migrate-status migrate-down-to migrate-reset migrate-create migrate-fix db-force-clean db-drop db-create db-drop-create








