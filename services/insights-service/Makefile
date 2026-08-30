# Variables
MIGRATIONS_PATH = ./migrations

# Load the .env file if it exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Migrations connect as the OWNER role, never the app role (ADR-0010 §2).
# DB_USER/DB_PASSWORD are the app credentials the SERVICE runs as and hold no
# DDL, so using them here would fail on the first CREATE TABLE.
DB_STRING=postgres://$(DB_OWNER_USER):$(DB_OWNER_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable
MIGRATE_COMMAND = migrate -path $(MIGRATIONS_PATH) -database "$(DB_STRING)"

.PHONY: build run dev test test-unit test-integration test-preview clean \
        migrate-up migrate-down migrate-status migrate-down-to migrate-reset \
        migrate-create migrate-fix docker-up docker-down

# --- build / run ---

build:
	@go build -o bin/insights-service cmd/server/main.go

run:
	@go run cmd/server/main.go

dev:
	@air

clean:
	@rm -rf bin/ tmp/

# --- tests ---

test:
	@go test -v ./... -cover

test-unit:
	@go test -v -short ./... -cover

test-integration:
	@go test -v -run Integration ./...

test-preview:
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out

# --- migrations (golang-migrate) ---

# up/down run the in-repo binary, so the common path needs no external tool
# and matches exactly what the compose init container will run after I-0040.
# The remaining targets still use the golang-migrate CLI — they are maintenance
# operations, not part of a deploy.
migrate-up:
	@go run ./cmd/migrate up

migrate-down:
	@go run ./cmd/migrate down

migrate-status:
	@$(MIGRATE_COMMAND) version

migrate-down-to:
	@if [ -z "$(VERSION)" ]; then \
		echo "Usage: make migrate-down-to VERSION=<version>"; \
		exit 1; \
	fi; \
	$(MIGRATE_COMMAND) down $(VERSION)

migrate-reset:
	@$(MIGRATE_COMMAND) drop

migrate-create:
	@if [ -z "$(NAME)" ]; then \
		echo "Usage: make migrate-create NAME=<migration_name>"; \
		exit 1; \
	fi; \
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $(NAME)

migrate-fix:
	@if [ -z "$(version)" ]; then \
		echo "Usage: make migrate-fix version=<version_number>"; \
		$(MIGRATE_COMMAND) version; \
		exit 1; \
	fi; \
	echo "Fixing dirty migration by setting version to $(version)..."; \
	$(MIGRATE_COMMAND) force $(version)

# --- docker (proxy to root docker-compose) ---

docker-up:
	@docker compose -f ../../docker-compose.yml up -d insights-service-db fireplace-consul fireplace-rabbitmq

docker-down:
	@docker compose -f ../../docker-compose.yml down
