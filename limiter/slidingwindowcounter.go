package limiter

import (
	"sync"
	"time"
)

// SlidingWindowCounter approximates a sliding window by keeping only two
// fixed-window counters (previous and current) and weighting the
// previous one by how much of it still overlaps the trailing window.
// O(1) memory per key, unlike SlidingWindowLog, at the cost of being an
// approximation (assumes uniform request distribution within a window).
type SlidingWindowCounter struct {
	limit Limit
	clock func() time.Time

	mu      sync.Mutex
	windows map[string]*counterWindow
}

type counterWindow struct {
	previous    int
	current     int
	windowStart time.Time
}

// NewSlidingWindowCounter creates a SlidingWindowCounter limiter enforcing limit.
func NewSlidingWindowCounter(limit Limit) *SlidingWindowCounter {
	return &SlidingWindowCounter{
		limit:   limit,
		clock:   time.Now,
		windows: make(map[string]*counterWindow),
	}
}

func (s *SlidingWindowCounter) Allow(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.clock()
	w, ok := s.windows[key]
	if !ok {
		s.windows[key] = &counterWindow{current: 1, windowStart: now}
		return true
	}

	elapsed := now.Sub(w.windowStart)
	if elapsed >= s.limit.Period {
		periods := elapsed / s.limit.Period
		if periods == 1 {
			w.previous = w.current
		} else {
			w.previous = 0
		}
		w.current = 0
		w.windowStart = w.windowStart.Add(periods * s.limit.Period)
		elapsed = now.Sub(w.windowStart)
	}

	weight := float64(s.limit.Period-elapsed) / float64(s.limit.Period)
	estimated := float64(w.previous)*weight + float64(w.current)

	if estimated >= float64(s.limit.Rate) {
		return false
	}
	w.current++
	return true
}
