// Package redislimiter is the distributed counterpart to package limiter:
// it enforces one rate limit shared across any number of app instances by
// keeping the counter state in Redis instead of process memory.
package redislimiter

import (
	"context"
	_ "embed"
	"fmt"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/omerfrkshn/ratelimiter-go/limiter"
)

//go:embed slidingwindow.lua
var slidingWindowScript string

// SlidingWindow is a sliding-window-log limiter (see limiter.SlidingWindowLog
// for the in-memory equivalent) backed by a Redis sorted set. The
// check-and-increment is a single Lua script, so it stays correct when
// many app instances call Allow concurrently against the same key.
type SlidingWindow struct {
	client    redis.Cmdable
	script    *redis.Script
	limit     limiter.Limit
	keyPrefix string
}

// NewSlidingWindow creates a distributed SlidingWindow limiter enforcing
// limit, sharing state through client.
func NewSlidingWindow(client redis.Cmdable, limit limiter.Limit) *SlidingWindow {
	return &SlidingWindow{
		client:    client,
		script:    redis.NewScript(slidingWindowScript),
		limit:     limit,
		keyPrefix: "ratelimit:",
	}
}

// Allow reports whether a request for key is permitted right now. Unlike
// limiter.Limiter, this can fail (e.g. Redis unreachable), so the error
// must be checked: callers should decide themselves whether a Redis
// outage should fail open or closed.
func (s *SlidingWindow) Allow(ctx context.Context, key string) (bool, error) {
	now := time.Now().UnixNano()
	windowStart := now - s.limit.Period.Nanoseconds()
	ttlSeconds := s.limit.Period.Seconds() + 1 // headroom over the window itself

	res, err := s.script.Run(ctx, s.client,
		[]string{s.keyPrefix + key},
		now, windowStart, s.limit.Rate, ttlSeconds, rand.Int63(),
	).Result()
	if err != nil {
		return false, fmt.Errorf("redislimiter: script failed: %w", err)
	}

	allowed, ok := res.(int64)
	if !ok {
		return false, fmt.Errorf("redislimiter: unexpected script result %T(%v)", res, res)
	}
	return allowed == 1, nil
}
