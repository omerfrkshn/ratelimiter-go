package limiter

import (
	"testing"
	"time"
)

func TestSlidingWindowLog_AllowsUpToRateWithinWindow(t *testing.T) {
	now := time.Unix(0, 0)
	sw := NewSlidingWindowLog(Limit{Rate: 3, Period: time.Second})
	sw.clock = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if !sw.Allow("a") {
			t.Fatalf("request %d: expected allow, got deny", i+1)
		}
	}
	if sw.Allow("a") {
		t.Fatal("4th request within window: expected deny")
	}
}

func TestSlidingWindowLog_ExpiredEntriesFreeUpCapacity(t *testing.T) {
	now := time.Unix(0, 0)
	sw := NewSlidingWindowLog(Limit{Rate: 2, Period: time.Second})
	sw.clock = func() time.Time { return now }

	sw.Allow("a") // t=0
	now = now.Add(500 * time.Millisecond)
	sw.Allow("a") // t=0.5, both still within last 1s
	if sw.Allow("a") {
		t.Fatal("3rd request: expected deny, window has 2 entries")
	}

	now = now.Add(600 * time.Millisecond) // t=1.1: first entry (t=0) now expired
	if !sw.Allow("a") {
		t.Fatal("expected allow: oldest entry should have fallen out of the window")
	}
}

func TestSlidingWindowLog_NoBoundaryBurst(t *testing.T) {
	// Unlike fixed window, a sliding window log must never allow more
	// than Rate requests in ANY trailing window of length Period.
	now := time.Unix(0, 0)
	sw := NewSlidingWindowLog(Limit{Rate: 2, Period: time.Second})
	sw.clock = func() time.Time { return now }

	sw.Allow("a")
	now = now.Add(900 * time.Millisecond)
	sw.Allow("a")

	now = now.Add(50 * time.Millisecond) // t=0.95s: still only 2 requests in last 1s
	if sw.Allow("a") {
		t.Fatal("expected deny: 2 requests already within trailing 1s window")
	}
}

func TestSlidingWindowLog_KeysAreIndependent(t *testing.T) {
	now := time.Unix(0, 0)
	sw := NewSlidingWindowLog(Limit{Rate: 1, Period: time.Second})
	sw.clock = func() time.Time { return now }

	if !sw.Allow("a") || !sw.Allow("b") {
		t.Fatal("expected independent keys to each get their own allowance")
	}
}
