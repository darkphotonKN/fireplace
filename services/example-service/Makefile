# example-service has no DB, so migrate-* targets are intentionally omitted.

# Load the .env file if it exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: build run dev test test-unit test-integration test-preview clean \
        docker-up docker-down

# --- build / run ---

build:
	@go build -o bin/example-service cmd/server/main.go

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

# --- docker (proxy to root docker-compose) ---

docker-up:
	@docker compose -f ../../docker-compose.yml up -d fireplace-consul fireplace-rabbitmq

docker-down:
	@docker compose -f ../../docker-compose.yml down
