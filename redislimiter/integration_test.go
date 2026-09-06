//go:build integration

// Integration tests require a real Redis reachable at REDIS_ADDR (default
// localhost:6379, see docker-compose.yml). Run with:
//
//	go test -tags=integration ./redislimiter/...
package redislimiter

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/omerfrkshn/ratelimiter-go/limiter"
)

func redisAddr() string {
	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		return addr
	}
	return "localhost:6379"
}

func newTestClient(t *testing.T) *redis.Client {
	t.Helper()
	client := redis.NewClient(&redis.Options{Addr: redisAddr()})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not reachable at %s (start it with `docker compose up -d`): %v", redisAddr(), err)
	}
	return client
}

func TestSlidingWindow_SingleInstance_EnforcesLimit(t *testing.T) {
	client := newTestClient(t)
	defer client.Close()

	key := fmt.Sprintf("test-single-%d", time.Now().UnixNano())
	rl := NewSlidingWindow(client, limiter.Limit{Rate: 5, Period: time.Minute})
	defer client.Del(context.Background(), "ratelimit:"+key)

	ctx := context.Background()
	allowed := 0
	for i := 0; i < 10; i++ {
		ok, err := rl.Allow(ctx, key)
		if err != nil {
			t.Fatalf("Allow: %v", err)
		}
		if ok {
			allowed++
		}
	}
	if allowed != 5 {
		t.Fatalf("expected 5 allowed, got %d", allowed)
	}
}

// TestSlidingWindow_TwoInstancesShareTheLimit is the core distributed-mode
// claim: two independent Redis clients (standing in for two app instances
// behind a load balancer) enforcing the same limit on the same key must
// together admit no more than Rate requests, not Rate-per-instance.
func TestSlidingWindow_TwoInstancesShareTheLimit(t *testing.T) {
	clientA := newTestClient(t)
	defer clientA.Close()
	clientB := newTestClient(t)
	defer clientB.Close()

	key := fmt.Sprintf("test-shared-%d", time.Now().UnixNano())
	const rate = 50
	limit := limiter.Limit{Rate: rate, Period: time.Minute}

	instanceA := NewSlidingWindow(clientA, limit)
	instanceB := NewSlidingWindow(clientB, limit)
	defer clientA.Del(context.Background(), "ratelimit:"+key)

	const requestsPerInstance = 100
	var wg sync.WaitGroup
	var mu sync.Mutex
	totalAllowed := 0

	fire := func(rl *SlidingWindow) {
		defer wg.Done()
		ctx := context.Background()
		for i := 0; i < requestsPerInstance; i++ {
			ok, err := rl.Allow(ctx, key)
			if err != nil {
				t.Errorf("Allow: %v", err)
				return
			}
			if ok {
				mu.Lock()
				totalAllowed++
				mu.Unlock()
			}
		}
	}

	wg.Add(2)
	go fire(instanceA)
	go fire(instanceB)
	wg.Wait()

	if totalAllowed != rate {
		t.Fatalf("expected exactly %d total admitted across both instances, got %d (limit not shared correctly)", rate, totalAllowed)
	}
}
