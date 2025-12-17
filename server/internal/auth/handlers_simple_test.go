package auth

import (
	"os"
	"testing"
)

// Test that development mode works without email service
func TestEmailService_DevelopmentMode(t *testing.T) {
	// Ensure no RESEND_API_KEY is set
	oldAPIKey := os.Getenv("RESEND_API_KEY")
	os.Unsetenv("RESEND_API_KEY")
	defer func() {
		if oldAPIKey != "" {
			os.Setenv("RESEND_API_KEY", oldAPIKey)
		}
	}()

	authService := NewAuthService("test-secret")
	handlers := NewAuthHandlers(nil, authService) // nil queries for this simple test

	// This should not fail even without email service configured
	err := handlers.sendMagicLinkEmail("test@example.com", "http://localhost:3000/verify?token=123", "login")
	if err != nil {
		t.Fatalf("expected no error in development mode, got %v", err)
	}
}

func TestEmailHelperFunctions(t *testing.T) {
	tests := []struct {
		purpose          string
		expectedTitle    string
		expectedButton   string
		expectedContains string
	}{
		{
			purpose:          "signup",
			expectedTitle:    "Welcome to PowHunter!",
			expectedButton:   "Verify Email & Complete Setup",
			expectedContains: "Thanks for joining",
		},
		{
			purpose:          "login",
			expectedTitle:    "Sign in to PowHunter",
			expectedButton:   "Sign In to PowHunter",
			expectedContains: "Click the button below to sign in",
		},
	}

	for _, tt := range tests {
		t.Run(tt.purpose, func(t *testing.T) {
			title := getEmailTitle(tt.purpose)
			if title != tt.expectedTitle {
				t.Errorf("getEmailTitle(%s) = %s, want %s", tt.purpose, title, tt.expectedTitle)
			}

			button := getButtonText(tt.purpose)
			if button != tt.expectedButton {
				t.Errorf("getButtonText(%s) = %s, want %s", tt.purpose, button, tt.expectedButton)
			}

			message := getEmailMessage(tt.purpose)
			if len(message) == 0 {
				t.Errorf("getEmailMessage(%s) returned empty string", tt.purpose)
			}
		})
	}
}

func TestGetFromEmail(t *testing.T) {
	// Test default
	oldFromEmail := os.Getenv("FROM_EMAIL")
	os.Unsetenv("FROM_EMAIL")
	defer func() {
		if oldFromEmail != "" {
			os.Setenv("FROM_EMAIL", oldFromEmail)
		}
	}()

	fromEmail := getFromEmail()
	if fromEmail != "noreply@powhunter.app" {
		t.Errorf("getFromEmail() = %s, want noreply@powhunter.app", fromEmail)
	}

	// Test with env var
	testEmail := "custom@example.com"
	os.Setenv("FROM_EMAIL", testEmail)
	fromEmail = getFromEmail()
	if fromEmail != testEmail {
		t.Errorf("getFromEmail() = %s, want %s", fromEmail, testEmail)
	}
}