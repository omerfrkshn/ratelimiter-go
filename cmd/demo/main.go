// Command demo runs a small HTTP server protected by the rate limiter
// middleware, so the library's behavior can be exercised with real
// requests (curl, vegeta, a browser, ...).
package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/omerfrkshn/ratelimiter-go/limiter"
	"github.com/omerfrkshn/ratelimiter-go/middleware"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	algorithm := flag.String("algorithm", "token-bucket", "fixed-window | sliding-log | sliding-counter | token-bucket")
	rate := flag.Int("rate", 100, "requests allowed per period")
	period := flag.Duration("period", time.Minute, "period duration, e.g. 60s")
	flag.Parse()

	limit := limiter.Limit{Rate: *rate, Period: *period}

	var l limiter.Limiter
	switch *algorithm {
	case "fixed-window":
		l = limiter.NewFixedWindow(limit)
	case "sliding-log":
		l = limiter.NewSlidingWindowLog(limit)
	case "sliding-counter":
		l = limiter.NewSlidingWindowCounter(limit)
	case "token-bucket":
		l = limiter.NewTokenBucket(limit)
	default:
		log.Fatalf("unknown algorithm %q", *algorithm)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	handler := middleware.RateLimit(middleware.Config{
		Limiter: l,
		KeyFunc: middleware.KeyByIP,
	})(mux)

	log.Printf("demo server listening on %s (algorithm=%s, limit=%d/%s)", *addr, *algorithm, *rate, *period)
	log.Fatal(http.ListenAndServe(*addr, handler))
}
