// Package middleware wraps a limiter.Limiter as net/http middleware,
// rejecting requests over the limit with 429 Too Many Requests.
package middleware

import (
	"net"
	"net/http"

	"github.com/omerfrkshn/ratelimiter-go/limiter"
)

// KeyFunc extracts the rate-limit key (user ID, API key, IP, ...) from a
// request. The middleware has no opinion on how the key is derived.
type KeyFunc func(r *http.Request) string

// Config controls RateLimit's behavior.
type Config struct {
	Limiter limiter.Limiter
	KeyFunc KeyFunc

	// Message is the response body sent with a 429. Defaults to
	// "rate limit exceeded" if empty.
	Message string
}

// RateLimit returns middleware that rejects requests with 429 once the
// key returned by cfg.KeyFunc has exceeded cfg.Limiter's limit.
func RateLimit(cfg Config) func(http.Handler) http.Handler {
	message := cfg.Message
	if message == "" {
		message = "rate limit exceeded"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := cfg.KeyFunc(r)
			if !cfg.Limiter.Allow(key) {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(message))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// KeyByRemoteAddr is a KeyFunc that uses the request's RemoteAddr
// (typically the caller's IP:port) as the rate-limit key.
func KeyByRemoteAddr(r *http.Request) string {
	return r.RemoteAddr
}

// KeyByIP is a KeyFunc that uses only the caller's IP (RemoteAddr with
// the port stripped) as the rate-limit key, so multiple connections or
// requests from the same client share one limit regardless of the
// ephemeral source port each connection happens to use.
func KeyByIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
