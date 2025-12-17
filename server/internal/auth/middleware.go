package auth

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	db "github.com/MattSilvaa/powhunter/internal/db/generated"
)

// AuthMiddleware handles JWT token validation for protected routes
type AuthMiddleware struct {
	AuthService *AuthService
	Queries     *db.Queries
}

// NewAuthMiddleware creates a new auth middleware instance
func NewAuthMiddleware(authService *AuthService, queries *db.Queries) *AuthMiddleware {
	return &AuthMiddleware{
		AuthService: authService,
		Queries:     queries,
	}
}

// RequireAuth middleware that validates JWT tokens
func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Check if header starts with "Bearer "
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		// Extract the token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate JWT token
		claims, err := m.AuthService.ValidateJWT(tokenString)
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		ctx := context.Background()

		// Verify session still exists in database
		session, err := m.Queries.GetUserSession(ctx, claims.SessionID)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "Session not found or expired", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Update session last used time
		if err := m.Queries.UpdateSessionLastUsed(ctx, session.SessionID); err != nil {
			// Log but don't fail the request
		}

		// Add claims to request context
		ctx = context.WithValue(r.Context(), "claims", claims)
		ctx = context.WithValue(ctx, "user_uuid", claims.UserUUID)
		ctx = context.WithValue(ctx, "session_id", claims.SessionID)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth middleware that adds user info to context if token is provided
// but doesn't require authentication
func (m *AuthMiddleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			// No auth provided, continue without user context
			next.ServeHTTP(w, r)
			return
		}

		// Extract the token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate JWT token
		claims, err := m.AuthService.ValidateJWT(tokenString)
		if err != nil {
			// Invalid token, continue without user context
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.Background()

		// Verify session still exists in database
		session, err := m.Queries.GetUserSession(ctx, claims.SessionID)
		if err != nil {
			// Session not found, continue without user context
			next.ServeHTTP(w, r)
			return
		}

		// Update session last used time
		if err := m.Queries.UpdateSessionLastUsed(ctx, session.SessionID); err != nil {
			// Log but don't fail the request
		}

		// Add claims to request context
		ctx = context.WithValue(r.Context(), "claims", claims)
		ctx = context.WithValue(ctx, "user_uuid", claims.UserUUID)
		ctx = context.WithValue(ctx, "session_id", claims.SessionID)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}