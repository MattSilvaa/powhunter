package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	db "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/resend/resend-go/v2"
)

// AuthHandlers contains the database queries and auth service
type AuthHandlers struct {
	Queries     *db.Queries
	AuthService *AuthService
}

// NewAuthHandlers creates a new auth handlers instance
func NewAuthHandlers(queries *db.Queries, authService *AuthService) *AuthHandlers {
	return &AuthHandlers{
		Queries:     queries,
		AuthService: authService,
	}
}

// SendMagicLinkRequest represents the request to send a magic link
type SendMagicLinkRequest struct {
	Email   string `json:"email"`
	Purpose string `json:"purpose"` // "login" or "signup"
}

// VerifyMagicLinkRequest represents the request to verify a magic link
type VerifyMagicLinkRequest struct {
	Token string `json:"token"`
}

// AuthResponse represents the response after successful authentication
type AuthResponse struct {
	Token     string         `json:"token"`
	User      db.User        `json:"user"`
	ExpiresAt time.Time      `json:"expires_at"`
}

// SendMagicLink creates and sends a magic link to the user's email
func (h *AuthHandlers) SendMagicLink(w http.ResponseWriter, r *http.Request) {
	var req SendMagicLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate email
	if req.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}

	// Set default purpose
	if req.Purpose == "" {
		req.Purpose = "login"
	}

	ctx := context.Background()
	
	// Find or create user
	user, err := h.Queries.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if err == sql.ErrNoRows && req.Purpose == "signup" {
			// Create new user for signup
			user, err = h.Queries.CreateUser(ctx, db.CreateUserParams{
				Email: req.Email,
				Phone: sql.NullString{},
			})
			if err != nil {
				http.Error(w, "Failed to create user", http.StatusInternalServerError)
				return
			}
		} else if err == sql.ErrNoRows {
			// User doesn't exist and this is a login attempt
			http.Error(w, "User not found", http.StatusNotFound)
			return
		} else {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}

	// Generate magic link token
	token, err := h.AuthService.GenerateMagicLinkToken()
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Store token in database
	_, err = h.Queries.CreateMagicLinkToken(ctx, db.CreateMagicLinkTokenParams{
		Token:     token,
		UserUuid:  user.Uuid,
		ExpiresAt: h.AuthService.GetMagicLinkExpiration(),
		Purpose:   req.Purpose,
	})
	if err != nil {
		http.Error(w, "Failed to store token", http.StatusInternalServerError)
		return
	}

	// Send magic link email (placeholder - implement actual email sending)
	magicLink := h.buildMagicLink(token, req.Purpose)
	if err := h.sendMagicLinkEmail(req.Email, magicLink, req.Purpose); err != nil {
		http.Error(w, "Failed to send email", http.StatusInternalServerError)
		return
	}

	// Clean up expired tokens
	h.Queries.CleanupExpiredTokens(ctx)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Magic link sent to your email",
	})
}

// VerifyMagicLink verifies a magic link token and returns a JWT
func (h *AuthHandlers) VerifyMagicLink(w http.ResponseWriter, r *http.Request) {
	var req VerifyMagicLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Token == "" {
		http.Error(w, "Token is required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Get and validate magic link token
	magicToken, err := h.Queries.GetMagicLinkToken(ctx, req.Token)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Mark token as used
	if err := h.Queries.UseMagicLinkToken(ctx, req.Token); err != nil {
		http.Error(w, "Failed to use token", http.StatusInternalServerError)
		return
	}

	// Get user
	user, err := h.Queries.GetUserByUUID(ctx, magicToken.UserUuid)
	if err != nil {
		http.Error(w, "User not found", http.StatusInternalServerError)
		return
	}

	// Mark email as verified if not already
	if !user.EmailVerified.Bool {
		if err := h.Queries.MarkEmailVerified(ctx, user.Uuid); err != nil {
			// Log but don't fail
		}
		// Update user struct for response
		user.EmailVerified = sql.NullBool{Bool: true, Valid: true}
		user.VerifiedAt = sql.NullTime{Time: time.Now(), Valid: true}
	}

	// Generate session ID
	sessionID, err := h.AuthService.GenerateSessionID()
	if err != nil {
		http.Error(w, "Failed to generate session", http.StatusInternalServerError)
		return
	}

	// Create session in database
	_, err = h.Queries.CreateUserSession(ctx, db.CreateUserSessionParams{
		SessionID: sessionID,
		UserUuid:  user.Uuid,
		ExpiresAt: h.AuthService.GetSessionExpiration(),
	})
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Generate JWT
	token, err := h.AuthService.CreateJWT(user.Uuid, user.Email, sessionID)
	if err != nil {
		http.Error(w, "Failed to create token", http.StatusInternalServerError)
		return
	}

	// Clean up expired sessions and tokens
	h.Queries.CleanupExpiredSessions(ctx)
	h.Queries.CleanupExpiredTokens(ctx)

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		Token:     token,
		User:      user,
		ExpiresAt: h.AuthService.GetSessionExpiration(),
	})
}

// Logout invalidates a user session
func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	// Get session ID from JWT claims (set by middleware)
	claims := r.Context().Value("claims").(*JWTClaims)
	if claims == nil {
		http.Error(w, "No session found", http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	// Delete session from database
	if err := h.Queries.DeleteUserSession(ctx, claims.SessionID); err != nil {
		http.Error(w, "Failed to logout", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Successfully logged out",
	})
}

// buildMagicLink constructs the magic link URL
func (h *AuthHandlers) buildMagicLink(token, purpose string) string {
	baseURL := os.Getenv("FRONTEND_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3000" // Default for development
	}

	// Build the verification URL
	u, _ := url.Parse(baseURL)
	u.Path = "/auth/verify"
	
	// Add query parameters
	params := url.Values{}
	params.Add("token", token)
	params.Add("purpose", purpose)
	u.RawQuery = params.Encode()

	return u.String()
}

// sendMagicLinkEmail sends the magic link via email
func (h *AuthHandlers) sendMagicLinkEmail(email, magicLink, purpose string) error {
	// Check if we have an API key for sending emails
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		// Fallback to logging for development
		fmt.Printf("Magic link for %s (%s): %s\n", email, purpose, magicLink)
		fmt.Printf("Set RESEND_API_KEY environment variable to enable email sending\n")
		return nil
	}

	// Use Resend to send the email
	client := resend.NewClient(apiKey)
	
	subject := "Your PowHunter Magic Link"
	if purpose == "signup" {
		subject = "Welcome to PowHunter - Verify Your Email"
	}

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>%s</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="text-align: center; margin-bottom: 30px;">
        <h1 style="color: #333;">PowHunter</h1>
    </div>
    
    <div style="background-color: #f8f9fa; padding: 30px; border-radius: 8px; margin-bottom: 30px;">
        <h2 style="color: #333; margin-top: 0;">%s</h2>
        <p style="color: #666; line-height: 1.6;">
            %s
        </p>
        
        <div style="text-align: center; margin: 30px 0;">
            <a href="%s" style="background-color: #007bff; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; display: inline-block; font-weight: bold;">
                %s
            </a>
        </div>
        
        <p style="color: #666; font-size: 14px; line-height: 1.6;">
            If the button doesn't work, you can copy and paste this link into your browser:
        </p>
        <p style="word-break: break-all; color: #007bff; font-size: 14px;">
            %s
        </p>
    </div>
    
    <div style="border-top: 1px solid #eee; padding-top: 20px; color: #999; font-size: 12px;">
        <p>This magic link will expire in 15 minutes for security reasons.</p>
        <p>If you didn't request this, you can safely ignore this email.</p>
    </div>
</body>
</html>`, 
		subject,
		getEmailTitle(purpose),
		getEmailMessage(purpose),
		magicLink,
		getButtonText(purpose),
		magicLink)

	params := &resend.SendEmailRequest{
		From:    getFromEmail(),
		To:      []string{email},
		Subject: subject,
		Html:    html,
	}

	_, err := client.Emails.Send(params)
	if err != nil {
		// Log the error but also fallback to console logging so development isn't blocked
		fmt.Printf("Failed to send email via Resend: %v\n", err)
		fmt.Printf("Magic link for %s (%s): %s\n", email, purpose, magicLink)
		return nil // Don't fail the request if email sending fails
	}

	fmt.Printf("Magic link email sent successfully to %s\n", email)
	return nil
}

func getFromEmail() string {
	fromEmail := os.Getenv("FROM_EMAIL")
	if fromEmail == "" {
		return "noreply@powhunter.app" // Default
	}
	return fromEmail
}

func getEmailTitle(purpose string) string {
	if purpose == "signup" {
		return "Welcome to PowHunter!"
	}
	return "Sign in to PowHunter"
}

func getEmailMessage(purpose string) string {
	if purpose == "signup" {
		return "Thanks for joining PowHunter! Click the button below to verify your email and complete your account setup. You'll then be able to manage your powder alert subscriptions."
	}
	return "Click the button below to sign in to your PowHunter account. You'll be able to view and manage your powder alert subscriptions."
}

func getButtonText(purpose string) string {
	if purpose == "signup" {
		return "Verify Email & Complete Setup"
	}
	return "Sign In to PowHunter"
}