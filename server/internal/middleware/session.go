package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/MattSilvaa/powhunter/internal/auth"
)

// userKey carries the authenticated user through the request context.
const userKey contextKey = "user"

// Authenticator resolves a session token to a user.
type Authenticator interface {
	Authenticate(ctx context.Context, sessionToken string) (auth.User, error)
}

// UserFromContext returns the authenticated user, and whether there was one.
func UserFromContext(ctx context.Context) (auth.User, bool) {
	user, ok := ctx.Value(userKey).(auth.User)
	return user, ok
}

// Session attaches the authenticated user to the request context when the
// request carries a valid session cookie. It never rejects a request on its
// own: routes that are readable both signed in and signed out stay usable, and
// RequireUser enforces authentication where it matters.
func Session(authenticator Authenticator, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := auth.TokenFromRequest(r)
			if token == "" {
				next.ServeHTTP(w, r)
				return
			}

			user, err := authenticator.Authenticate(r.Context(), token)
			if err != nil {
				// An unknown or expired cookie is an ordinary signed-out request,
				// not an error worth logging. Anything else is a real fault.
				if !errors.Is(err, auth.ErrNoSession) {
					logger.Error("failed to resolve session",
						"error", err,
						"request_id", RequestIDFromContext(r.Context()),
					)
				}

				next.ServeHTTP(w, r)

				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, user)))
		})
	}
}

// RequireUser rejects requests that Session did not authenticate.
func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserFromContext(r.Context()); !ok {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"UNAUTHENTICATED","message":"Please sign in to continue"}`))

			return
		}

		next.ServeHTTP(w, r)
	})
}
