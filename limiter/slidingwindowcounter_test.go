package limiter

import (
	"testing"
	"time"
)

func TestSlidingWindowCounter_AllowsUpToRateWithinWindow(t *testing.T) {
	now := time.Unix(0, 0)
	sw := NewSlidingWindowCounter(Limit{Rate: 3, Period: time.Second})
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

func TestSlidingWindowCounter_WeightsPreviousWindow(t *testing.T) {
	// Rate=10, Period=1s. Fill window 1 completely (10 requests), then
	// move to 50% into window 2: weighted estimate is
	// previous(10)*0.5 + current(0) = 5, so 5 more requests should be
	// allowed before the 6th is denied.
	now := time.Unix(0, 0)
	sw := NewSlidingWindowCounter(Limit{Rate: 10, Period: time.Second})
	sw.clock = func() time.Time { return now }

	for i := 0; i < 10; i++ {
		if !sw.Allow("a") {
			t.Fatalf("window 1 request %d: expected allow", i+1)
		}
	}

	now = now.Add(1500 * time.Millisecond) // 500ms into window 2
	allowed := 0
	for i := 0; i < 10; i++ {
		if sw.Allow("a") {
			allowed++
		}
	}
	if allowed != 5 {
		t.Fatalf("expected 5 allowed in weighted window, got %d", allowed)
	}
}

func TestSlidingWindowCounter_FullWindowGapResetsPrevious(t *testing.T) {
	now := time.Unix(0, 0)
	sw := NewSlidingWindowCounter(Limit{Rate: 2, Period: time.Second})
	sw.clock = func() time.Time { return now }

	sw.Allow("a")
	sw.Allow("a")

	now = now.Add(3 * time.Second) // more than one full period elapsed
	if !sw.Allow("a") {
		t.Fatal("expected allow: previous window should have fully decayed")
	}
}

func TestSlidingWindowCounter_KeysAreIndependent(t *testing.T) {
	now := time.Unix(0, 0)
	sw := NewSlidingWindowCounter(Limit{Rate: 1, Period: time.Second})
	sw.clock = func() time.Time { return now }

	if !sw.Allow("a") || !sw.Allow("b") {
		t.Fatal("expected independent keys to each get their own allowance")
	}
}
