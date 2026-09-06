package limiter

import (
	"sync"
	"testing"
	"time"
)

// TestConcurrentAccess_NoRace hammers each algorithm from many goroutines
// across a handful of keys. It exists to be run under `go test -race`:
// the only thing it asserts is that the race detector finds nothing, and
// that no call panics. Correctness under concurrency is a property of
// the mutex in each implementation, not of this test.
func TestConcurrentAccess_NoRace(t *testing.T) {
	limit := Limit{Rate: 50, Period: 50 * time.Millisecond}

	limiters := map[string]Limiter{
		"FixedWindow":          NewFixedWindow(limit),
		"SlidingWindowLog":     NewSlidingWindowLog(limit),
		"SlidingWindowCounter": NewSlidingWindowCounter(limit),
		"TokenBucket":          NewTokenBucket(limit),
	}

	keys := []string{"user-1", "user-2", "user-3"}
	const goroutines = 50
	const requestsPerGoroutine = 200

	for name, l := range limiters {
		name, l := name, l
		t.Run(name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(goroutines)
			for g := 0; g < goroutines; g++ {
				go func(g int) {
					defer wg.Done()
					key := keys[g%len(keys)]
					for i := 0; i < requestsPerGoroutine; i++ {
						l.Allow(key)
					}
				}(g)
			}
			wg.Wait()
		})
	}
}

// TestConcurrentAccess_RespectsLimit checks that, even under concurrent
// access, the number of allowed requests for a single key never exceeds
// what a sequential run would allow at time zero (all goroutines racing
// to consume the same initial allowance).
func TestConcurrentAccess_RespectsLimit(t *testing.T) {
	const rate = 100
	limit := Limit{Rate: rate, Period: time.Minute} // long period: no refill during the test

	newLimiters := func() map[string]Limiter {
		return map[string]Limiter{
			"FixedWindow":          NewFixedWindow(limit),
			"SlidingWindowLog":     NewSlidingWindowLog(limit),
			"SlidingWindowCounter": NewSlidingWindowCounter(limit),
			"TokenBucket":          NewTokenBucket(limit),
		}
	}

	for name, l := range newLimiters() {
		name, l := name, l
		t.Run(name, func(t *testing.T) {
			const goroutines = 300
			var wg sync.WaitGroup
			var mu sync.Mutex
			allowed := 0

			wg.Add(goroutines)
			for g := 0; g < goroutines; g++ {
				go func() {
					defer wg.Done()
					if l.Allow("shared-key") {
						mu.Lock()
						allowed++
						mu.Unlock()
					}
				}()
			}
			wg.Wait()

			if allowed != rate {
				t.Fatalf("%s: expected exactly %d allowed under race, got %d (mutex not preventing over-admission)", name, rate, allowed)
			}
		})
	}
}
