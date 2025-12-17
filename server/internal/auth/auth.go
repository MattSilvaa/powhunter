package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	MagicLinkTokenLength = 32
	MagicLinkExpiration  = 15 * time.Minute
	SessionExpiration    = 24 * 7 * time.Hour // 7 days
)

// JWTClaims represents the claims in a JWT token
type JWTClaims struct {
	UserUUID  uuid.UUID `json:"user_uuid"`
	Email     string    `json:"email"`
	SessionID string    `json:"session_id"`
	jwt.RegisteredClaims
}

// AuthService handles authentication operations
type AuthService struct {
	jwtSecret []byte
}

// NewAuthService creates a new authentication service
func NewAuthService(jwtSecret string) *AuthService {
	return &AuthService{
		jwtSecret: []byte(jwtSecret),
	}
}

// GenerateMagicLinkToken generates a cryptographically secure random token
func (a *AuthService) GenerateMagicLinkToken() (string, error) {
	bytes := make([]byte, MagicLinkTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateSessionID generates a unique session identifier
func (a *AuthService) GenerateSessionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// CreateJWT creates a JWT token for the user session
func (a *AuthService) CreateJWT(userUUID uuid.UUID, email, sessionID string) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserUUID:  userUUID,
		Email:     email,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(SessionExpiration)),
			Subject:   userUUID.String(),
			Issuer:    "powhunter",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.jwtSecret)
}

// ValidateJWT validates and parses a JWT token
func (a *AuthService) ValidateJWT(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// GetMagicLinkExpiration returns when a magic link token should expire
func (a *AuthService) GetMagicLinkExpiration() time.Time {
	return time.Now().Add(MagicLinkExpiration)
}

// GetSessionExpiration returns when a session should expire
func (a *AuthService) GetSessionExpiration() time.Time {
	return time.Now().Add(SessionExpiration)
}