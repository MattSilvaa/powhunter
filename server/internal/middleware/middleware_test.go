package middleware_test

import (
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
	assert.Equal(t, "198.51.100.7", middleware.ClientIP(req, true))
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
