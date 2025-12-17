package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAuthService_GenerateMagicLinkToken(t *testing.T) {
	authService := NewAuthService("test-secret")

	token, err := authService.GenerateMagicLinkToken()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty token")
	}

	if len(token) != MagicLinkTokenLength*2 { // hex encoding doubles length
		t.Fatalf("expected token length %d, got %d", MagicLinkTokenLength*2, len(token))
	}

	// Generate another token and ensure they're different
	token2, err := authService.GenerateMagicLinkToken()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if token == token2 {
		t.Fatal("expected different tokens on subsequent calls")
	}
}

func TestAuthService_GenerateSessionID(t *testing.T) {
	authService := NewAuthService("test-secret")

	sessionID, err := authService.GenerateSessionID()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if sessionID == "" {
		t.Fatal("expected non-empty session ID")
	}

	if len(sessionID) != 32 { // 16 bytes hex encoded = 32 characters
		t.Fatalf("expected session ID length 32, got %d", len(sessionID))
	}

	// Generate another session ID and ensure they're different
	sessionID2, err := authService.GenerateSessionID()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if sessionID == sessionID2 {
		t.Fatal("expected different session IDs on subsequent calls")
	}
}

func TestAuthService_CreateAndValidateJWT(t *testing.T) {
	authService := NewAuthService("test-secret-key")

	userUUID := uuid.New()
	email := "test@example.com"
	sessionID := "test-session-id"

	// Create JWT
	token, err := authService.CreateJWT(userUUID, email, sessionID)
	if err != nil {
		t.Fatalf("expected no error creating JWT, got %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty JWT token")
	}

	// Validate JWT
	claims, err := authService.ValidateJWT(token)
	if err != nil {
		t.Fatalf("expected no error validating JWT, got %v", err)
	}

	if claims.UserUUID != userUUID {
		t.Fatalf("expected UserUUID %v, got %v", userUUID, claims.UserUUID)
	}

	if claims.Email != email {
		t.Fatalf("expected Email %s, got %s", email, claims.Email)
	}

	if claims.SessionID != sessionID {
		t.Fatalf("expected SessionID %s, got %s", sessionID, claims.SessionID)
	}

	if claims.Subject != userUUID.String() {
		t.Fatalf("expected Subject %s, got %s", userUUID.String(), claims.Subject)
	}

	if claims.Issuer != "powhunter" {
		t.Fatalf("expected Issuer 'powhunter', got %s", claims.Issuer)
	}
}

func TestAuthService_ValidateJWT_InvalidToken(t *testing.T) {
	authService := NewAuthService("test-secret-key")

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"invalid format", "invalid-token"},
		{"random string", "this.is.not.a.jwt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := authService.ValidateJWT(tt.token)
			if err == nil {
				t.Fatal("expected error for invalid token, got nil")
			}
		})
	}
}

func TestAuthService_ValidateJWT_WrongSecret(t *testing.T) {
	authService1 := NewAuthService("secret1")
	authService2 := NewAuthService("secret2")

	userUUID := uuid.New()
	email := "test@example.com"
	sessionID := "test-session-id"

	// Create JWT with first service
	token, err := authService1.CreateJWT(userUUID, email, sessionID)
	if err != nil {
		t.Fatalf("expected no error creating JWT, got %v", err)
	}

	// Try to validate with second service (different secret)
	_, err = authService2.ValidateJWT(token)
	if err == nil {
		t.Fatal("expected error validating JWT with wrong secret, got nil")
	}
}

func TestAuthService_ExpirationMethods(t *testing.T) {
	authService := NewAuthService("test-secret")

	magicLinkExp := authService.GetMagicLinkExpiration()
	sessionExp := authService.GetSessionExpiration()

	now := time.Now()

	// Magic link expiration should be ~15 minutes from now
	expectedMagicLink := now.Add(MagicLinkExpiration)
	if magicLinkExp.Sub(expectedMagicLink) > time.Minute {
		t.Fatalf("magic link expiration is too far from expected time")
	}

	// Session expiration should be ~7 days from now
	expectedSession := now.Add(SessionExpiration)
	if sessionExp.Sub(expectedSession) > time.Minute {
		t.Fatalf("session expiration is too far from expected time")
	}
}