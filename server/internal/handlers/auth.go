package handlers

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/MattSilvaa/powhunter/internal/auth"
	"github.com/MattSilvaa/powhunter/internal/middleware"
	"github.com/MattSilvaa/powhunter/internal/notify"
	"github.com/MattSilvaa/powhunter/internal/validate"
)

// AuthService is the behaviour the auth handlers need, kept as an interface so
// the handlers can be tested without a database.
type AuthService interface {
	IssueLoginToken(ctx context.Context, email string) (string, error)
	Redeem(ctx context.Context, loginToken string) (string, time.Time, error)
	Logout(ctx context.Context, sessionToken string) error
	SetCookie(w http.ResponseWriter, token string, expiresAt time.Time)
	ClearCookie(w http.ResponseWriter)
}

// AuthHandler serves the magic-link login endpoints.
type AuthHandler struct {
	auth    AuthService
	mailer  notify.Mailer
	baseURL string
	logger  *slog.Logger
}

// NewAuthHandler builds the login endpoints.
func NewAuthHandler(service AuthService, mailer notify.Mailer, baseURL string, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{auth: service, mailer: mailer, baseURL: baseURL, logger: logger}
}

type requestLinkRequest struct {
	Email string `json:"email"`
}

type callbackRequest struct {
	Token string `json:"token"`
}

type userResponse struct {
	Email         string `json:"email"`
	Phone         string `json:"phone,omitempty"`
	EmailVerified bool   `json:"emailVerified"`
}

// RequestLink emails a single-use login link.
//
// The response is deliberately identical whether or not the address has an
// account, and whether or not the mail actually went out. Reporting "no such
// user" here would turn this endpoint into a way to test which addresses are
// registered.
func (h *AuthHandler) RequestLink(w http.ResponseWriter, r *http.Request) {
	var req requestLinkRequest
	if err := decodeJSON(r, &req); err != nil {
		sendDecodeError(w, err)
		return
	}

	email, err := validate.Email(req.Email)
	if err != nil {
		sendErrorResponse(w, "INVALID_EMAIL", "Please enter a valid email address", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), handlerTimeout)
	defer cancel()

	if err := h.sendLoginLink(ctx, email); err != nil {
		// Log the real reason, tell the caller nothing: see the note above.
		h.logger.Error("failed to send login link",
			"error", err,
			"request_id", middleware.RequestIDFromContext(r.Context()),
		)
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "If that address has an account, a sign-in link is on its way.",
	})
}

func (h *AuthHandler) sendLoginLink(ctx context.Context, email string) error {
	token, err := h.auth.IssueLoginToken(ctx, email)
	if err != nil {
		return fmt.Errorf("issuing login token: %w", err)
	}

	link := fmt.Sprintf("%s/login?token=%s", h.baseURL, url.QueryEscape(token))

	body := fmt.Sprintf(`
		<h2>Sign in to Powhunter</h2>
		<p>Click the link below to sign in. It works once and expires in %d minutes.</p>
		<p><a href="%s">Sign in to Powhunter</a></p>
		<p>If you did not request this, you can ignore this email.</p>
	`, int(auth.LoginTokenTTL.Minutes()), html.EscapeString(link))

	if err := h.mailer.Send(ctx, notify.Email{
		To:      []string{email},
		Subject: "Your Powhunter sign-in link",
		HTML:    body,
	}); err != nil {
		return fmt.Errorf("sending login email: %w", err)
	}

	return nil
}

// Callback exchanges a login token for a session cookie.
func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	var req callbackRequest
	if err := decodeJSON(r, &req); err != nil {
		sendDecodeError(w, err)
		return
	}

	if req.Token == "" {
		sendErrorResponse(w, "INVALID_TOKEN", "This sign-in link is not valid", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), handlerTimeout)
	defer cancel()

	sessionToken, expiresAt, err := h.auth.Redeem(ctx, req.Token)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidToken) {
			sendErrorResponse(
				w,
				"INVALID_TOKEN",
				"This sign-in link has expired or has already been used",
				http.StatusUnauthorized,
			)

			return
		}

		h.logger.Error("failed to redeem login token",
			"error", err,
			"request_id", middleware.RequestIDFromContext(r.Context()),
		)
		sendErrorResponse(w, "INTERNAL_ERROR", "Could not sign you in", http.StatusInternalServerError)

		return
	}

	h.auth.SetCookie(w, sessionToken, expiresAt)
	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// Logout revokes the current session.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), handlerTimeout)
	defer cancel()

	if err := h.auth.Logout(ctx, auth.TokenFromRequest(r)); err != nil {
		h.logger.Error("failed to revoke session",
			"error", err,
			"request_id", middleware.RequestIDFromContext(r.Context()),
		)
	}

	// The cookie is cleared either way: a caller asking to sign out must end up
	// signed out locally even if the revoking write failed.
	h.auth.ClearCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// Me returns the signed-in user, or 401.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		sendErrorResponse(w, "UNAUTHENTICATED", "Please sign in to continue", http.StatusUnauthorized)
		return
	}

	writeJSON(w, http.StatusOK, userResponse{
		Email:         user.Email,
		Phone:         user.Phone,
		EmailVerified: user.EmailVerified,
	})
}
