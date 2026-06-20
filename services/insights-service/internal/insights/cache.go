package insights

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache is the narrow Redis surface the insights service depends on. Per DIP the
// consumer (this package) owns the abstraction; per ISP it exposes only the
// handful of operations insights needs rather than the full *redis.Client. The
// concrete *redis.Client injected at SetupServices satisfies this directly, so
// no adapter is required.
//
// Nothing calls these yet — the seam is wired so cache-backed logic can be added
// later. Widen or trim this method set as the actual commands are written.
type Cache interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}
