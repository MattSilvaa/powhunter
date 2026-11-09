package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MattSilvaa/powhunter/internal/middleware"
	"github.com/stretchr/testify/assert"
)

func TestCrossOriginGuard(t *testing.T) {
	guard := middleware.CrossOriginGuard(middleware.NewCORSConfig("https://powhunter.app", true))(okHandler())

	cases := map[string]struct {
		method  string
		headers map[string]string
		want    int
	}{
		"the web app itself": {
			method:  http.MethodPost,
			headers: map[string]string{"Origin": "https://powhunter.app", "Sec-Fetch-Site": "cross-site"},
			want:    http.StatusOK,
		},
		// Production: powhunter.app calling api.powhunter.app.
		"the web app on the same site": {
			method:  http.MethodPost,
			headers: map[string]string{"Origin": "https://powhunter.app", "Sec-Fetch-Site": "same-site"},
			want:    http.StatusOK,
		},
		// A SameSite=Lax cookie still rides along from any sibling subdomain.
		"another subdomain of the same site": {
			method:  http.MethodPost,
			headers: map[string]string{"Origin": "https://evil.powhunter.app", "Sec-Fetch-Site": "same-site"},
			want:    http.StatusForbidden,
		},
		// A form or no-cors fetch from another site carries a SameSite=None
		// cookie and skips the CORS preflight.
		"a cross-site form post": {
			method:  http.MethodPost,
			headers: map[string]string{"Origin": "https://evil.example", "Sec-Fetch-Site": "cross-site"},
			want:    http.StatusForbidden,
		},
		"an old browser from a foreign origin": {
			method:  http.MethodDelete,
			headers: map[string]string{"Origin": "https://evil.example"},
			want:    http.StatusForbidden,
		},
		"same origin": {
			method:  http.MethodPatch,
			headers: map[string]string{"Sec-Fetch-Site": "same-origin"},
			want:    http.StatusOK,
		},
		"a non-browser client": {
			method: http.MethodPost,
			want:   http.StatusOK,
		},
		"a cross-site read": {
			method:  http.MethodGet,
			headers: map[string]string{"Origin": "https://evil.example", "Sec-Fetch-Site": "cross-site"},
			want:    http.StatusOK,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/api/user/alerts", nil)
			for header, value := range tc.headers {
				req.Header.Set(header, value)
			}

			rr := httptest.NewRecorder()
			guard.ServeHTTP(rr, req)
			assert.Equal(t, tc.want, rr.Code)
		})
	}
}
