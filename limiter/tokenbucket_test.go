package limiter

import (
	"testing"
	"time"
)

func TestTokenBucket_AllowsFullBurstImmediately(t *testing.T) {
	now := time.Unix(0, 0)
	tb := NewTokenBucket(Limit{Rate: 5, Period: time.Second})
	tb.clock = func() time.Time { return now }

	for i := 0; i < 5; i++ {
		if !tb.Allow("a") {
			t.Fatalf("burst request %d: expected allow", i+1)
		}
	}
	if tb.Allow("a") {
		t.Fatal("6th immediate request: expected deny, bucket should be empty")
	}
}

func TestTokenBucket_RefillsOverTime(t *testing.T) {
	now := time.Unix(0, 0)
	tb := NewTokenBucket(Limit{Rate: 10, Period: time.Second}) // 10 tokens/sec
	tb.clock = func() time.Time { return now }

	for i := 0; i < 10; i++ {
		tb.Allow("a")
	}
	if tb.Allow("a") {
		t.Fatal("expected deny: bucket drained")
	}

	now = now.Add(500 * time.Millisecond) // should refill ~5 tokens
	allowed := 0
	for i := 0; i < 10; i++ {
		if tb.Allow("a") {
			allowed++
		}
	}
	if allowed != 5 {
		t.Fatalf("expected 5 refilled tokens to be allowed, got %d", allowed)
	}
}

func TestTokenBucket_NeverExceedsCapacity(t *testing.T) {
	now := time.Unix(0, 0)
	tb := NewTokenBucket(Limit{Rate: 3, Period: time.Second})
	tb.clock = func() time.Time { return now }

	tb.Allow("a") // consume 1, 2 left

	now = now.Add(time.Hour) // huge idle gap, tokens must cap at capacity
	allowed := 0
	for i := 0; i < 10; i++ {
		if tb.Allow("a") {
			allowed++
		}
	}
	if allowed != 3 {
		t.Fatalf("expected bucket to cap at capacity 3, got %d allowed", allowed)
	}
}

func TestTokenBucket_KeysAreIndependent(t *testing.T) {
	now := time.Unix(0, 0)
	tb := NewTokenBucket(Limit{Rate: 1, Period: time.Second})
	tb.clock = func() time.Time { return now }

	if !tb.Allow("a") || !tb.Allow("b") {
		t.Fatal("expected independent keys to each get their own bucket")
	}
}
