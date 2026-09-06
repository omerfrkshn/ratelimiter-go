package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/omerfrkshn/ratelimiter-go/limiter"
)

func TestRateLimit_AllowsWithinLimitThenReturns429(t *testing.T) {
	cfg := Config{
		Limiter: limiter.NewFixedWindow(limiter.Limit{Rate: 2, Period: time.Minute}),
		KeyFunc: func(r *http.Request) string { return "fixed-key" },
	}
	handler := RateLimit(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rec.Code)
		}
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("3rd request: expected 429, got %d", rec.Code)
	}
}

func TestRateLimit_KeysAreIsolatedByKeyFunc(t *testing.T) {
	cfg := Config{
		Limiter: limiter.NewFixedWindow(limiter.Limit{Rate: 1, Period: time.Minute}),
		KeyFunc: KeyByRemoteAddr,
	}
	handler := RateLimit(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "10.0.0.1:1234"
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "10.0.0.2:5678"

	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if rec1.Code != http.StatusOK || rec2.Code != http.StatusOK {
		t.Fatalf("expected both distinct-IP requests to be allowed, got %d and %d", rec1.Code, rec2.Code)
	}
}
