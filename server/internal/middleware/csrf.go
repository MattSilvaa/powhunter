package middleware

import (
	"net/http"
	"net/url"
)

// CrossOriginGuard rejects state-changing requests a browser sent on behalf of
// a site that is not on the CORS allowlist.
//
// The session cookie is SameSite=None in production, because the web app and
// the API are on different sites. That means a page on any site can make the
// victim's browser send a POST carrying the cookie: a form or a no-cors fetch
// with a text/plain body skips the CORS preflight, so the allowlist alone never
// gets a say. This closes that gap the same way Go 1.25's
// http.CrossOriginProtection does, using the Sec-Fetch-Site and Origin headers
// browsers attach and pages cannot forge.
//
// Requests without either header did not come from a browser, so there is no
// ambient cookie to abuse and they are let through.
func CrossOriginGuard(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSafeMethod(r.Method) || crossOriginAllowed(config, r) {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"CROSS_ORIGIN_REJECTED","message":"Cross-origin request rejected"}`))
		})
	}
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func crossOriginAllowed(config CORSConfig, r *http.Request) bool {
	origin := r.Header.Get("Origin")

	// The web app's own origin, or another one deliberately allowlisted.
	if origin != "" && config.allows(origin) {
		return true
	}

	switch r.Header.Get("Sec-Fetch-Site") {
	case "same-origin", "none":
		// "none" is a user-initiated navigation, such as a typed URL.
		return true
	case "":
		// An older browser that does not send Sec-Fetch-Site. Fall back to
		// comparing Origin with the host the request was sent to.
		if origin == "" {
			return true
		}

		parsed, err := url.Parse(origin)

		return err == nil && parsed.Host == r.Host
	default:
		// same-site or cross-site, from an origin that is not allowlisted.
		return false
	}
}
