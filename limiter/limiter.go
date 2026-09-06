// Package limiter provides in-memory rate limiting algorithms sharing a
// common interface, so callers can swap the strategy without changing
// call sites.
package limiter

import "time"

// Limiter decides whether a request identified by key is allowed to
// proceed right now. Implementations are safe for concurrent use.
type Limiter interface {
	// Allow reports whether a request for key is permitted under the
	// configured limit, and records the request if it is.
	Allow(key string) bool
}

// Limit describes "Rate requests per Period" for a single key.
type Limit struct {
	Rate   int
	Period time.Duration
}
