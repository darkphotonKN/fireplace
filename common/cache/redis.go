// Package cache provides the shared Redis client setup for fireplace services.
//
// This file only wires the connection — it intentionally exposes no domain
// commands. Each service injects the returned *redis.Client and writes its own
// Get/Set/etc. against it (or wraps it behind a service-local interface).
//
// Mirrors the broker.Connect style: read the config from env in your main.go,
// call Connect, and `defer client.Close()`.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config holds Redis connection settings. Populate from env in main.go, e.g.:
//
//	cache.Config{
//	    Addr:     commonhelpers.GetEnvString("REDIS_ADDR", "localhost:6380"),
//	    Password: commonhelpers.GetEnvString("REDIS_PASSWORD", ""),
//	    DB:       0,
//	}
//
// The default addr is localhost:6380 — the host port the root docker-compose
// maps to the fireplace-redis container's 6379.
type Config struct {
	Addr     string // host:port, e.g. "localhost:6380"
	Password string // empty for the local dev instance
	DB       int    // logical DB number, usually 0
}

// Connect builds a Redis client and verifies it with a bounded PING so a dead
// Redis surfaces at startup instead of on the first command. The ping is capped
// at 5s regardless of the passed context so a hung Redis can't block boot
// indefinitely.
//
// The returned *redis.Client is safe for concurrent use and pools connections
// internally — create ONE per service and inject it; don't open one per call.
// Call client.Close() on shutdown.
func Connect(ctx context.Context, cfg Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		// Close the half-open client so we don't leak its pool on a failed boot.
		_ = client.Close()
		return nil, fmt.Errorf("cache: connect redis at %s: %w", cfg.Addr, err)
	}

	return client, nil
}
