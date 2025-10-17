package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/resend/resend-go/v2"
)

type ContactHandler struct{}

func NewContactHandler() (*ContactHandler, error) {
	return &ContactHandler{}, nil
}

type ContactRequest struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
}

func (h *ContactHandler) HandleContact(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorResponse(w, "METHOD_NOT_ALLOWED", "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	setSecurityHeaders(w)

	var req ContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if strings.TrimSpace(req.Name) == "" {
		sendErrorResponse(w, "MISSING_NAME", "Name is required", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Email) == "" {
		sendErrorResponse(w, "MISSING_EMAIL", "Email is required", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Message) == "" {
		sendErrorResponse(w, "MISSING_MESSAGE", "Message is required", http.StatusBadRequest)
		return
	}

	// Basic email validation
	if !strings.Contains(req.Email, "@") || !strings.Contains(req.Email, ".") {
		sendErrorResponse(w, "INVALID_EMAIL", "Invalid email address", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Log the contact message
	log.Printf("Contact form submission from %s (%s): %s", req.Name, req.Email, req.Message)

	// TODO: Send email notification
	// For now, we'll just log it and optionally write to a file
	if err := h.recordContactMessage(ctx, req); err != nil {
		log.Printf("Failed to record contact message: %v", err)
		sendErrorResponse(w, "INTERNAL_ERROR", "Failed to process contact message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Thank you for contacting us! We'll get back to you soon.",
	})

	if err != nil {
		log.Printf("Failed to write response: %v", err)
		return
	}
}

func (h *ContactHandler) recordContactMessage(ctx context.Context, req ContactRequest) error {
	// Log to file
	contactLogPath := os.Getenv("CONTACT_LOG_PATH")
	if contactLogPath == "" {
		contactLogPath = "/tmp/powhunter_contacts.log"
	}

	f, err := os.OpenFile(contactLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open contact log file: %w", err)
	}
	defer f.Close()

	timestamp := time.Now().Format(time.RFC3339)
	logEntry := fmt.Sprintf("[%s] Name: %s | Email: %s | Message: %s\n",
		timestamp, req.Name, req.Email, req.Message)

	if _, err := f.WriteString(logEntry); err != nil {
		return fmt.Errorf("failed to write to contact log: %w", err)
	}

	// Send email notification
	if err := h.sendContactEmail(ctx, req); err != nil {
		log.Printf("Warning: Failed to send contact email notification: %v", err)
		// We don't return error here to not block the request if email fails
	}

	return nil
}

func (h *ContactHandler) sendContactEmail(ctx context.Context, req ContactRequest) error {
	// Get Resend API key from environment
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		log.Println("RESEND_API_KEY not configured, skipping email send")
		return nil
	}

	client := resend.NewClient(apiKey)

	// Construct email body
	htmlBody := fmt.Sprintf(`
		<h2>New Contact Form Submission</h2>
		<p><strong>From:</strong> %s (%s)</p>
		<p><strong>Submitted:</strong> %s</p>
		<h3>Message:</h3>
		<p>%s</p>
		<hr>
		<p><em>This email was sent from the Powhunter contact form.</em></p>
	`, req.Name, req.Email, time.Now().Format("January 2, 2006 at 3:04 PM MST"), strings.ReplaceAll(req.Message, "\n", "<br>"))

	params := &resend.SendEmailRequest{
		From:    "Powhunter <noreply@powhunter.app>",
		To:      []string{"support@powhunter.app"},
		ReplyTo: req.Email,
		Subject: fmt.Sprintf("Contact Form Submission from %s", req.Name),
		Html:    htmlBody,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Contact email sent to support@powhunter.app from %s (%s) - ID: %s", req.Name, req.Email, sent.Id)
	return nil
}
