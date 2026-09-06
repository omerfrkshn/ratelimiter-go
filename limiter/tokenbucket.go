package limiter

import (
	"sync"
	"time"
)

// TokenBucket allows a burst of up to limit.Rate requests immediately,
// then refills at a steady rate of limit.Rate tokens per limit.Period.
// The only one of the four algorithms designed to tolerate bursts rather
// than smooth them out.
type TokenBucket struct {
	limit Limit
	clock func() time.Time

	mu      sync.Mutex
	buckets map[string]*bucketState
}

type bucketState struct {
	tokens     float64
	lastRefill time.Time
}

// NewTokenBucket creates a TokenBucket limiter with capacity limit.Rate,
// refilling at limit.Rate tokens per limit.Period.
func NewTokenBucket(limit Limit) *TokenBucket {
	return &TokenBucket{
		limit:   limit,
		clock:   time.Now,
		buckets: make(map[string]*bucketState),
	}
}

func (t *TokenBucket) Allow(key string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.clock()
	capacity := float64(t.limit.Rate)
	refillRate := capacity / t.limit.Period.Seconds() // tokens per second

	b, ok := t.buckets[key]
	if !ok {
		b = &bucketState{tokens: capacity, lastRefill: now}
		t.buckets[key] = b
	} else {
		elapsed := now.Sub(b.lastRefill).Seconds()
		b.tokens += elapsed * refillRate
		if b.tokens > capacity {
			b.tokens = capacity
		}
		b.lastRefill = now
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
