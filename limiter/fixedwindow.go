package limiter

import (
	"sync"
	"time"
)

// FixedWindow counts requests in a fixed-size time window that resets
// entirely once it elapses. Simplest and cheapest algorithm, but allows
// up to 2x the configured rate at window boundaries (a burst just before
// a window ends, followed by another just after it starts).
type FixedWindow struct {
	limit Limit
	clock func() time.Time

	mu      sync.Mutex
	windows map[string]*fixedWindowState
}

type fixedWindowState struct {
	windowStart time.Time
	count       int
}

// NewFixedWindow creates a FixedWindow limiter enforcing limit.
func NewFixedWindow(limit Limit) *FixedWindow {
	return &FixedWindow{
		limit:   limit,
		clock:   time.Now,
		windows: make(map[string]*fixedWindowState),
	}
}

func (f *FixedWindow) Allow(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := f.clock()
	s, ok := f.windows[key]
	if !ok || now.Sub(s.windowStart) >= f.limit.Period {
		f.windows[key] = &fixedWindowState{windowStart: now, count: 1}
		return true
	}

	if s.count >= f.limit.Rate {
		return false
	}
	s.count++
	return true
}
