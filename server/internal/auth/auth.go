// Package auth implements passwordless, magic-link authentication.
//
// Two kinds of secret are issued here, and neither is ever stored in the clear:
// a short-lived, single-use login token that is emailed to the address claiming
// an account, and a longer-lived session token that is returned as a cookie.
// The database holds only a SHA-256 hash of each, so a leak of the tables does
// not hand an attacker usable credentials.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/google/uuid"
)

const (
	// tokenBytes is the entropy behind both token kinds. 256 bits is far beyond
	// guessable, which matters because a login token is a bearer credential.
	tokenBytes = 32

	// LoginTokenTTL bounds how long an emailed link stays usable. Short enough
	// that a link left in an inbox is not a standing key, long enough to survive
	// slow mail delivery.
	LoginTokenTTL = 15 * time.Minute

	// SessionTTL is how long a login lasts without activity.
	SessionTTL = 30 * 24 * time.Hour

	// sessionRefreshInterval throttles the expiry-extending write. Without it
	// every authenticated request would issue an UPDATE.
	sessionRefreshInterval = 24 * time.Hour

	// CookieName is the session cookie.
	CookieName = "powhunter_session"
)

// ErrInvalidToken reports a login token that is unknown, expired, or already
// redeemed. The three cases are deliberately indistinguishable to the caller.
var ErrInvalidToken = errors.New("invalid or expired login token")

// ErrNoSession reports a request that carries no usable session.
var ErrNoSession = errors.New("no active session")

// Queries is the slice of the generated query set this package needs. Keeping
// it narrow lets tests substitute a fake without a database.
type Queries interface {
	GetOrCreateUserByEmail(ctx context.Context, email string) (dbgen.User, error)
	CreateLoginToken(ctx context.Context, arg dbgen.CreateLoginTokenParams) (dbgen.LoginToken, error)
	ConsumeLoginToken(ctx context.Context, tokenHash string) (uuid.UUID, error)
	CreateSession(ctx context.Context, arg dbgen.CreateSessionParams) (dbgen.Session, error)
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (dbgen.GetSessionByTokenHashRow, error)
	TouchSession(ctx context.Context, arg dbgen.TouchSessionParams) error
	DeleteSession(ctx context.Context, tokenHash string) error
	DeleteExpiredSessions(ctx context.Context) error
	DeleteExpiredLoginTokens(ctx context.Context) error
	MarkEmailVerified(ctx context.Context, userUUID uuid.UUID) error
}

// User is the authenticated identity attached to a request.
type User struct {
	UUID          uuid.UUID
	Email         string
	Phone         string
	EmailVerified bool
}

// Service issues and validates login tokens and sessions.
type Service struct {
	queries   Queries
	secure    bool
	crossSite bool
	now       func() time.Time
}

// NewService builds a Service. Pass secure=true in production so the session
// cookie is only ever sent over TLS, and crossSite=true when the browser app is
// served from a different site than this API: a browser silently discards a
// SameSite=Lax cookie that arrives on a cross-site request, so the login would
// appear to succeed and leave the caller signed out.
func NewService(queries Queries, secure, crossSite bool) *Service {
	return &Service{queries: queries, secure: secure, crossSite: crossSite, now: time.Now}
}

// newToken returns a URL-safe bearer token and its storage hash.
func newToken() (string, string, error) {
	buf := make([]byte, tokenBytes)
	if _, readErr := rand.Read(buf); readErr != nil {
		return "", "", fmt.Errorf("generating token: %w", readErr)
	}

	token := base64.RawURLEncoding.EncodeToString(buf)

	return token, HashToken(token), nil
}

// HashToken returns the value stored for a token. SHA-256 is the right choice
// here, unlike for a password: these tokens carry full random entropy, so the
// slow hashing that defends a guessable secret buys nothing.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// IssueLoginToken finds or creates the account for email and returns a token to
// send to that address. The account is created here so signing up and signing in
// are the same action from the caller's point of view.
func (s *Service) IssueLoginToken(ctx context.Context, email string) (string, error) {
	user, err := s.queries.GetOrCreateUserByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("resolving user for %q: %w", email, err)
	}

	token, hash, err := newToken()
	if err != nil {
		return "", err
	}

	if _, storeErr := s.queries.CreateLoginToken(ctx, dbgen.CreateLoginTokenParams{
		UserUuid:  user.Uuid,
		TokenHash: hash,
		ExpiresAt: s.now().Add(LoginTokenTTL),
	}); storeErr != nil {
		return "", fmt.Errorf("storing login token: %w", storeErr)
	}

	return token, nil
}

// Redeem exchanges a login token for a session token. The login token is
// consumed in the process and cannot be replayed.
func (s *Service) Redeem(ctx context.Context, loginToken string) (string, time.Time, error) {
	userUUID, err := s.queries.ConsumeLoginToken(ctx, HashToken(loginToken))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", time.Time{}, ErrInvalidToken
		}

		return "", time.Time{}, fmt.Errorf("consuming login token: %w", err)
	}

	// Redeeming a link proves control of the address it was sent to.
	if verifyErr := s.queries.MarkEmailVerified(ctx, userUUID); verifyErr != nil {
		return "", time.Time{}, fmt.Errorf("marking email verified: %w", verifyErr)
	}

	token, hash, err := newToken()
	if err != nil {
		return "", time.Time{}, err
	}

	expiresAt := s.now().Add(SessionTTL)

	if _, createErr := s.queries.CreateSession(ctx, dbgen.CreateSessionParams{
		UserUuid:  userUUID,
		TokenHash: hash,
		ExpiresAt: expiresAt,
	}); createErr != nil {
		return "", time.Time{}, fmt.Errorf("creating session: %w", createErr)
	}

	return token, expiresAt, nil
}

// Authenticate resolves a session token to its user, extending the session when
// it is far enough from issue that a write is worth it.
func (s *Service) Authenticate(ctx context.Context, sessionToken string) (User, error) {
	if sessionToken == "" {
		return User{}, ErrNoSession
	}

	hash := HashToken(sessionToken)

	row, err := s.queries.GetSessionByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return User{}, ErrNoSession
		}

		return User{}, fmt.Errorf("looking up session: %w", err)
	}

	if time.Until(row.ExpiresAt) < SessionTTL-sessionRefreshInterval {
		// A failed refresh must not fail the request: the session is still valid
		// for now, and the next request will try again. The error is deliberately
		// dropped rather than returned.
		_ = s.queries.TouchSession(ctx, dbgen.TouchSessionParams{
			TokenHash: hash,
			ExpiresAt: s.now().Add(SessionTTL),
		})
	}

	return userFromRow(row), nil
}

// Logout revokes a session.
func (s *Service) Logout(ctx context.Context, sessionToken string) error {
	if sessionToken == "" {
		return nil
	}

	if err := s.queries.DeleteSession(ctx, HashToken(sessionToken)); err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}

	return nil
}

// Reap removes expired sessions and login tokens.
func (s *Service) Reap(ctx context.Context) error {
	if err := s.queries.DeleteExpiredSessions(ctx); err != nil {
		return fmt.Errorf("deleting expired sessions: %w", err)
	}

	if err := s.queries.DeleteExpiredLoginTokens(ctx); err != nil {
		return fmt.Errorf("deleting expired login tokens: %w", err)
	}

	return nil
}

// SetCookie writes the session cookie.
func (s *Service) SetCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:  CookieName,
		Value: token,
		Path:  "/",
		// The cookie must be unreadable to scripts: an XSS bug should not also
		// be a session theft.
		HttpOnly: true,
		Secure:   s.cookieSecure(),
		SameSite: s.cookieSameSite(),
		Expires:  expiresAt,
	})
}

// ClearCookie expires the session cookie. Its attributes must match those
// SetCookie wrote, or the browser treats it as a different cookie and keeps the
// original.
func (s *Service) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   s.cookieSecure(),
		SameSite: s.cookieSameSite(),
		MaxAge:   -1,
	})
}

// cookieSameSite picks the strictest policy the deployment can actually use. A
// browser app on another site only receives the cookie under SameSite=None.
func (s *Service) cookieSameSite() http.SameSite {
	if s.crossSite {
		return http.SameSiteNoneMode
	}

	return http.SameSiteLaxMode
}

// cookieSecure reports whether to mark the cookie Secure. SameSite=None forces
// it: browsers reject that combination without Secure, which would leave the
// cookie unset rather than merely less strict.
func (s *Service) cookieSecure() bool {
	return s.secure || s.crossSite
}

// TokenFromRequest reads the session token from the request cookie.
func TokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return ""
	}

	return cookie.Value
}

func userFromRow(row dbgen.GetSessionByTokenHashRow) User {
	return User{
		UUID:          row.UserUuid,
		Email:         row.Email,
		Phone:         row.Phone.String,
		EmailVerified: row.EmailVerifiedAt.Valid,
	}
}
