package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	// visitorTTL is how long an idle visitor's bucket is retained.
	visitorTTL = 10 * time.Minute
	// sweepInterval is how often expired visitors are discarded.
	sweepInterval = time.Minute
)

// RateLimitConfig describes one bucket of allowance.
type RateLimitConfig struct {
	// RequestsPerSecond is the sustained rate allowed per client.
	RequestsPerSecond float64
	// Burst is how many requests may arrive at once.
	Burst int
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiter tracks a token bucket per client address.
//
// This is per-process state, so it does not coordinate across replicas. That is
// adequate for the current single-instance deployment; a shared store would be
// needed before scaling horizontally.
type RateLimiter struct {
	config     RateLimitConfig
	trustProxy bool

	mu       sync.Mutex
	visitors map[string]*visitor
	stop     chan struct{}
}

// NewRateLimiter creates a limiter and starts its cleanup goroutine.
func NewRateLimiter(config RateLimitConfig, trustProxy bool) *RateLimiter {
	limiter := &RateLimiter{
		config:     config,
		trustProxy: trustProxy,
		visitors:   make(map[string]*visitor),
		stop:       make(chan struct{}),
	}

	go limiter.sweep()

	return limiter
}

// Close stops the cleanup goroutine.
func (l *RateLimiter) Close() {
	close(l.stop)
}

func (l *RateLimiter) sweep() {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-l.stop:
			return
		case now := <-ticker.C:
			l.mu.Lock()

			for key, v := range l.visitors {
				if now.Sub(v.lastSeen) > visitorTTL {
					delete(l.visitors, key)
				}
			}

			l.mu.Unlock()
		}
	}
}

func (l *RateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	v, ok := l.visitors[key]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(rate.Limit(l.config.RequestsPerSecond), l.config.Burst)}
		l.visitors[key] = v
	}

	v.lastSeen = time.Now()

	return v.limiter.Allow()
}

// Middleware rejects requests from a client that exceeds its allowance.
func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if l.allow(ClientIP(r, l.trustProxy)) {
			next.ServeHTTP(w, r)

			return
		}

		retryAfter := 1
		if l.config.RequestsPerSecond > 0 {
			retryAfter = int(1/l.config.RequestsPerSecond) + 1
		}

		w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)

		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":   "RATE_LIMITED",
			"message": "Too many requests. Please slow down and try again shortly.",
		})
	})
}
