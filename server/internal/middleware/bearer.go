package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// bearerPrefix is the scheme the Authorization header must use.
const bearerPrefix = "Bearer "

// RequireBearerToken gates a route on a shared secret.
//
// It exists for /metrics, which was served to anyone who asked: request counts,
// route names and database pool statistics are an operational map of the
// service, and a scraper is the only thing that needs them. An empty token
// leaves the route open, which config.ValidateWeb refuses to allow in
// production.
func RequireBearerToken(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if token == "" {
			return next
		}

		expected := []byte(token)

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, bearerPrefix) {
				unauthorized(w)
				return
			}

			// Constant time, so a caller cannot recover the token by measuring
			// how long the comparison takes.
			presented := []byte(strings.TrimPrefix(header, bearerPrefix))
			if subtle.ConstantTimeCompare(presented, expected) != 1 {
				unauthorized(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"UNAUTHENTICATED","message":"A valid token is required"}`))
}
