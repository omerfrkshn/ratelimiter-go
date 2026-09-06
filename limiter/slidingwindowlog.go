package limiter

import (
	"sync"
	"time"
)

// SlidingWindowLog keeps a timestamp per request and counts how many
// fall within the last Period. Exact (no boundary burst issue), but
// memory grows with the number of requests allowed per key per window.
type SlidingWindowLog struct {
	limit Limit
	clock func() time.Time

	mu   sync.Mutex
	logs map[string][]time.Time
}

// NewSlidingWindowLog creates a SlidingWindowLog limiter enforcing limit.
func NewSlidingWindowLog(limit Limit) *SlidingWindowLog {
	return &SlidingWindowLog{
		limit: limit,
		clock: time.Now,
		logs:  make(map[string][]time.Time),
	}
}

func (s *SlidingWindowLog) Allow(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.clock()
	cutoff := now.Add(-s.limit.Period)

	log := s.logs[key]
	i := 0
	for i < len(log) && !log[i].After(cutoff) {
		i++
	}
	log = log[i:]

	if len(log) >= s.limit.Rate {
		s.logs[key] = log
		return false
	}

	log = append(log, now)
	s.logs[key] = log
	return true
}
