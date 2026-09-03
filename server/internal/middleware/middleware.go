// Package middleware provides the HTTP middleware chain used by the API.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"time"
)

type contextKey string

// requestIDKey carries the per-request identifier through the request context.
const requestIDKey contextKey = "requestID"

// RequestIDHeader is the header used to accept and echo a request identifier.
const RequestIDHeader = "X-Request-ID"

// requestIDBytes is the length of a generated request identifier.
const requestIDBytes = 16

// maxRequestIDLength bounds an inbound identifier so it cannot bloat logs.
const maxRequestIDLength = 128

// RequestIDFromContext returns the request identifier, or an empty string.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

func newRequestID() string {
	buf := make([]byte, requestIDBytes)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand failing is not a reason to drop the request; fall back to
		// a time-based identifier, which is still useful for correlation.
		return "ts-" + time.Now().UTC().Format("20060102T150405.000000000")
	}

	return hex.EncodeToString(buf)
}

// RequestID attaches an identifier to every request, reusing an inbound one so
// a trace survives the reverse proxy, and echoes it back to the caller.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if id == "" || len(id) > maxRequestIDLength {
			id = newRequestID()
		}

		w.Header().Set(RequestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
	})
}

// statusRecorder captures the status code and response size for logging.
type statusRecorder struct {
	http.ResponseWriter
	status  int
	written int
	wrote   bool
}

func (s *statusRecorder) WriteHeader(status int) {
	if s.wrote {
		return
	}

	s.status = status
	s.wrote = true
	s.ResponseWriter.WriteHeader(status)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if !s.wrote {
		s.WriteHeader(http.StatusOK)
	}

	n, err := s.ResponseWriter.Write(b)
	s.written += n

	return n, err //nolint:wrapcheck // pass through the underlying writer's error
}

// Status reports the status code written, defaulting to 200.
func (s *statusRecorder) Status() int {
	if s.status == 0 {
		return http.StatusOK
	}

	return s.status
}

// Recover keeps a panicking handler from taking down the process, and records
// the stack so the failure is visible rather than silent.
func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}

				logger.Error("recovered from panic in handler",
					"panic", recovered,
					"method", r.Method,
					"path", r.URL.Path,
					"request_id", RequestIDFromContext(r.Context()),
					"stack", string(debug.Stack()),
				)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"INTERNAL_ERROR","message":"Something went wrong"}`))
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// Logging emits one structured line per request, correlated by request ID.
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &statusRecorder{ResponseWriter: w}

			next.ServeHTTP(recorder, r)

			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.Status(),
				"bytes", recorder.written,
				"duration_ms", time.Since(start).Milliseconds(),
				"remote_ip", ClientIP(r, false),
				"request_id", RequestIDFromContext(r.Context()),
			)
		})
	}
}

// SecurityHeaders applies the headers every response should carry. Previously
// each handler called a helper by hand, so a new handler silently lost them.
func SecurityHeaders(isProduction bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := w.Header()
			header.Set("X-Content-Type-Options", "nosniff")
			header.Set("X-Frame-Options", "DENY")
			header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			header.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

			if isProduction {
				header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// NoStore marks responses that carry user data as uncacheable.
func NoStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// MaxBytes caps the request body so a large upload cannot be read into memory.
func MaxBytes(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

// ClientIP returns the address to attribute a request to. The proxy header is
// only trusted when the deployment actually sits behind a proxy, otherwise any
// caller could spoof it to escape rate limiting.
func ClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			// The left-most entry is the original client.
			for i := range len(forwarded) {
				if forwarded[i] == ',' {
					return trimSpace(forwarded[:i])
				}
			}

			return trimSpace(forwarded)
		}

		if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
			return trimSpace(realIP)
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

func trimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}

	return s[start:end]
}

// Chain applies middleware so the first argument is the outermost layer.
func Chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	return handler
}
