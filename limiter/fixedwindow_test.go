package limiter

import (
	"testing"
	"time"
)

func TestFixedWindow_AllowsUpToRateWithinWindow(t *testing.T) {
	now := time.Unix(0, 0)
	fw := NewFixedWindow(Limit{Rate: 3, Period: time.Second})
	fw.clock = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if !fw.Allow("a") {
			t.Fatalf("request %d: expected allow, got deny", i+1)
		}
	}
	if fw.Allow("a") {
		t.Fatal("4th request within window: expected deny, got allow")
	}
}

func TestFixedWindow_ResetsOnNewWindow(t *testing.T) {
	now := time.Unix(0, 0)
	fw := NewFixedWindow(Limit{Rate: 1, Period: time.Second})
	fw.clock = func() time.Time { return now }

	if !fw.Allow("a") {
		t.Fatal("first request: expected allow")
	}
	if fw.Allow("a") {
		t.Fatal("second request same window: expected deny")
	}

	now = now.Add(time.Second)
	if !fw.Allow("a") {
		t.Fatal("first request of new window: expected allow")
	}
}

func TestFixedWindow_KeysAreIndependent(t *testing.T) {
	now := time.Unix(0, 0)
	fw := NewFixedWindow(Limit{Rate: 1, Period: time.Second})
	fw.clock = func() time.Time { return now }

	if !fw.Allow("a") {
		t.Fatal("key a: expected allow")
	}
	if !fw.Allow("b") {
		t.Fatal("key b: expected allow, keys must not share state")
	}
}

func TestFixedWindow_BoundaryBurstIsAllowedByDesign(t *testing.T) {
	// Documents the known weakness: up to 2x rate can pass across a
	// window boundary. This is expected fixed-window behavior, not a bug.
	now := time.Unix(0, 0)
	fw := NewFixedWindow(Limit{Rate: 2, Period: time.Second})
	fw.clock = func() time.Time { return now }

	fw.Allow("a")
	fw.Allow("a") // 2 requests at the end of window 1

	now = now.Add(time.Second) // start of window 2
	if !fw.Allow("a") || !fw.Allow("a") {
		t.Fatal("expected 2 more allowed requests in the new window")
	}
}
