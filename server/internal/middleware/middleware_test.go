package middleware_test

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MattSilvaa/powhunter/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestCORSOnlyEchoesAllowedOrigins(t *testing.T) {
	config := middleware.NewCORSConfig("https://powhunter.app,https://staging.powhunter.app", true)
	handler := middleware.CORS(config)(okHandler())

	t.Run("allowed origin is echoed with credentials", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/resorts", nil)
		req.Header.Set("Origin", "https://powhunter.app")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, "https://powhunter.app", rr.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "true", rr.Header().Get("Access-Control-Allow-Credentials"))
		assert.Contains(t, rr.Header().Values("Vary"), "Origin")
	})

	t.Run("unknown origin gets no allow header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/resorts", nil)
		req.Header.Set("Origin", "https://evil.example")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
		assert.Empty(t, rr.Header().Get("Access-Control-Allow-Credentials"))
	})

	t.Run("preflight short-circuits with no content", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/alerts", nil)
		req.Header.Set("Origin", "https://powhunter.app")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)
		assert.NotEmpty(t, rr.Header().Get("Access-Control-Max-Age"))
		// Editing an alert or the profile is a PATCH; a preflight that omits it
		// blocks those requests from the cross-site web app.
		assert.Contains(t, rr.Header().Get("Access-Control-Allow-Methods"), "PATCH")
	})
}

// Browsers reject a wildcard origin paired with credentials, which silently
// broke cookie-bearing requests in development.
func TestCORSNeverPairsWildcardWithCredentials(t *testing.T) {
	handler := middleware.CORS(middleware.NewCORSConfig("*", true))(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/resorts", nil)
	req.Header.Set("Origin", "https://anywhere.example")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "*", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Credentials"))
}

func TestRecoverKeepsTheProcessAlive(t *testing.T) {
	panicking := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("boom")
	})
	handler := middleware.Recover(discardLogger())(panicking)

	rr := httptest.NewRecorder()
	require.NotPanics(t, func() {
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/resorts", nil))
	})

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "INTERNAL_ERROR")
}

func TestRateLimiterRejectsBeyondBurst(t *testing.T) {
	limiter := middleware.NewRateLimiter(
		middleware.RateLimitConfig{RequestsPerSecond: 1, Burst: 2},
		false,
	)
	defer limiter.Close()

	handler := limiter.Middleware(okHandler())

	statuses := make([]int, 0, 3)

	for range 3 {
		req := httptest.NewRequest(http.MethodPost, "/api/alerts", nil)
		req.RemoteAddr = "203.0.113.10:1234"
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		statuses = append(statuses, rr.Code)
	}

	assert.Equal(t, []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests}, statuses)
}

func TestRateLimiterIsPerClient(t *testing.T) {
	limiter := middleware.NewRateLimiter(
		middleware.RateLimitConfig{RequestsPerSecond: 1, Burst: 1},
		false,
	)
	defer limiter.Close()

	handler := limiter.Middleware(okHandler())

	first := httptest.NewRecorder()
	reqA := httptest.NewRequest(http.MethodPost, "/api/alerts", nil)
	reqA.RemoteAddr = "203.0.113.10:1234"
	handler.ServeHTTP(first, reqA)

	second := httptest.NewRecorder()
	reqB := httptest.NewRequest(http.MethodPost, "/api/alerts", nil)
	reqB.RemoteAddr = "203.0.113.11:1234"
	handler.ServeHTTP(second, reqB)

	assert.Equal(t, http.StatusOK, first.Code)
	assert.Equal(t, http.StatusOK, second.Code, "a different client must have its own allowance")
}

// An untrusted proxy header would otherwise let any caller escape rate limiting
// by spoofing a new address on every request.
func TestClientIPIgnoresProxyHeaderWhenUntrusted(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/resorts", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	req.Header.Set("X-Forwarded-For", "198.51.100.7, 203.0.113.1")

	assert.Equal(t, "203.0.113.10", middleware.ClientIP(req, false))
}

// Render's proxy appends to X-Forwarded-For, so the left-most entry is whatever
// the client sent. Trusting it let a caller pick a fresh address per request and
// never hit a rate limit.
func TestClientIPBehindAProxyCannotBeSpoofed(t *testing.T) {
	newRequest := func(headers map[string]string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/api/resorts", nil)
		req.RemoteAddr = "10.0.0.5:1234"
		for name, value := range headers {
			req.Header.Set(name, value)
		}

		return req
	}

	cases := map[string]struct {
		headers map[string]string
		want    string
	}{
		"edge header wins over a forged forwarded-for": {
			headers: map[string]string{
				"CF-Connecting-IP": "198.51.100.7",
				"X-Forwarded-For":  "1.2.3.4, 198.51.100.7",
			},
			want: "198.51.100.7",
		},
		"right-most forwarded-for entry is the one our proxy added": {
			headers: map[string]string{"X-Forwarded-For": "1.2.3.4, 198.51.100.7"},
			want:    "198.51.100.7",
		},
		"junk falls back to the connection address": {
			headers: map[string]string{"CF-Connecting-IP": "not-an-ip", "X-Forwarded-For": "also junk"},
			want:    "10.0.0.5",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, middleware.ClientIP(newRequest(tc.headers), true))
		})
	}
}

// A flood of distinct addresses must neither grow the table without bound nor
// escape limiting: once full, newcomers share one bucket.
func TestRateLimiterStaysBoundedUnderManyClients(t *testing.T) {
	// Slow enough that no bucket refills during the test, however loaded the
	// machine running it is.
	limiter := middleware.NewRateLimiter(middleware.RateLimitConfig{RequestsPerSecond: 0.001, Burst: 1}, false)
	defer limiter.Close()

	limiter.SetMaxVisitors(3)
	handler := limiter.Middleware(okHandler())

	statuses := make([]int, 0, 6)

	for i := range 6 {
		req := httptest.NewRequest(http.MethodPost, "/api/alerts", nil)
		req.RemoteAddr = fmt.Sprintf("203.0.113.%d:1234", i)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		statuses = append(statuses, rr.Code)
	}

	assert.LessOrEqual(t, limiter.VisitorCount(), 4, "three clients plus the shared overflow bucket")
	// The first three get their own allowance; the fourth takes the overflow
	// bucket's only token, and every later newcomer is refused.
	assert.Equal(t, []int{200, 200, 200, 200, 429, 429}, statuses)
}

func TestMaxBytesRejectsOversizedBodies(t *testing.T) {
	handler := middleware.MaxBytes(16)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(strings.Repeat("a", 1024)))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, rr.Code)
}

func TestRequestIDIsAttachedAndEchoed(t *testing.T) {
	var seen string

	handler := middleware.RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = middleware.RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("generates one when absent", func(t *testing.T) {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/resorts", nil))

		assert.NotEmpty(t, seen)
		assert.Equal(t, seen, rr.Header().Get(middleware.RequestIDHeader))
	})

	t.Run("reuses an inbound one so traces survive the proxy", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/resorts", nil)
		req.Header.Set(middleware.RequestIDHeader, "abc123")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, "abc123", seen)
		assert.Equal(t, "abc123", rr.Header().Get(middleware.RequestIDHeader))
	})
}

func TestSecurityHeadersAreAlwaysApplied(t *testing.T) {
	handler := middleware.SecurityHeaders(true)(okHandler())
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/resorts", nil))

	assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rr.Header().Get("X-Frame-Options"))
	assert.NotEmpty(t, rr.Header().Get("Strict-Transport-Security"))
	assert.NotEmpty(t, rr.Header().Get("Content-Security-Policy"))
}
