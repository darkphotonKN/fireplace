# syntax=docker/dockerfile:1
#
# Build context is the REPO ROOT, not this directory:
#   docker build -f services/api-gateway/Dockerfile .
#
# The module's go.mod carries `replace .../common => ../../common`, so the build
# needs common/ alongside this service but does NOT need the go.work workspace.
# GOWORK=off makes that explicit — with the workspace on, Go would demand every
# service listed in go.work be present in the image.

FROM golang:1.25-alpine AS build
ENV GOWORK=off CGO_ENABLED=0
WORKDIR /src

# Dependency layer first: these change far less often than source, so an edit to
# a .go file does not re-download the module cache.
COPY common/go.mod common/go.sum ./common/
COPY services/api-gateway/go.mod services/api-gateway/go.sum ./services/api-gateway/
RUN --mount=type=cache,target=/go/pkg/mod \
    cd services/api-gateway && go mod download

COPY common/ ./common/
COPY services/api-gateway/ ./services/api-gateway/

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd services/api-gateway && \
    go build -trimpath -ldflags="-s -w" -o /out/app ./cmd && \
    go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 app
WORKDIR /app

COPY --from=build /out/app  /app/app
COPY --from=build /out/migrate /app/migrate
# Migrations ship WITH the binary that applies them — cmd/migrate resolves
# file://migrations relative to this WORKDIR.
COPY services/api-gateway/migrations/ /app/migrations/

USER app
ENTRYPOINT ["/app/app"]
