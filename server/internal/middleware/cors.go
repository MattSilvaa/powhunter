package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// corsMaxAge is how long a browser may cache a preflight response.
const corsMaxAge = 10 * time.Minute

// CORSConfig describes which origins may call the API.
type CORSConfig struct {
	// AllowedOrigins is an exact-match allowlist. A single "*" entry allows any
	// origin, which is only safe because credentials are then refused.
	AllowedOrigins []string
	// AllowCredentials permits cookies. It is ignored for wildcard origins,
	// since browsers reject that combination outright.
	AllowCredentials bool
}

// NewCORSConfig parses a comma-separated origin list.
func NewCORSConfig(origins string, allowCredentials bool) CORSConfig {
	var allowed []string

	for _, origin := range strings.Split(origins, ",") {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			allowed = append(allowed, trimmed)
		}
	}

	return CORSConfig{AllowedOrigins: allowed, AllowCredentials: allowCredentials}
}

func (c CORSConfig) isWildcard() bool {
	return len(c.AllowedOrigins) == 1 && c.AllowedOrigins[0] == "*"
}

// allows reports whether the given origin may call the API.
func (c CORSConfig) allows(origin string) bool {
	if origin == "" {
		return false
	}

	if c.isWildcard() {
		return true
	}

	for _, allowed := range c.AllowedOrigins {
		if strings.EqualFold(allowed, origin) {
			return true
		}
	}

	return false
}

// CORS echoes an allow-origin header only for origins on the allowlist, rather
// than asserting one unconditionally. Credentials are never paired with a
// wildcard, a combination browsers reject and which silently broke cookies.
func CORS(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// The response varies by Origin even when the origin is rejected, so
			// caches must not serve one origin's response to another.
			w.Header().Add("Vary", "Origin")

			if config.allows(origin) {
				if config.isWildcard() {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					w.Header().Set("Access-Control-Allow-Origin", origin)

					if config.AllowCredentials {
						w.Header().Set("Access-Control-Allow-Credentials", "true")
					}
				}
			}

			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, "+RequestIDHeader)
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(int(corsMaxAge.Seconds())))
				w.WriteHeader(http.StatusNoContent)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
