package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/MattSilvaa/powhunter/internal/notify"
	"github.com/MattSilvaa/powhunter/internal/validate"
)

type ContactHandler struct {
	mailer  notify.Mailer
	support string
}

func NewContactHandler(mailer notify.Mailer, supportEmail string) (*ContactHandler, error) {
	return &ContactHandler{mailer: mailer, support: supportEmail}, nil
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
	if err := decodeJSON(r, &req); err != nil {
		sendDecodeError(w, err)
		return
	}

	name, err := validate.Text("name", req.Name, validate.MaxNameLength)
	if err != nil {
		sendErrorResponse(w, "MISSING_NAME", "Please enter your name", http.StatusBadRequest)
		return
	}

	message, err := validate.Text("message", req.Message, validate.MaxMessageLength)
	if err != nil {
		sendErrorResponse(w, "MISSING_MESSAGE", "Please enter a message", http.StatusBadRequest)
		return
	}

	email, err := validate.Email(req.Email)
	if err != nil {
		sendErrorResponse(w, "INVALID_EMAIL", "Please enter a valid email address", http.StatusBadRequest)
		return
	}

	req.Name = name
	req.Email = email
	req.Message = message

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// TODO: Send email notification
	// For now, we'll just log it and optionally write to a file
	if err := h.recordContactMessage(ctx, req); err != nil {
		log.Printf("Failed to record contact message: %v", err)
		sendErrorResponse(w, "INTERNAL_ERROR", "Failed to process contact message", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(map[string]string{
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
	// Construct email body
	// Every interpolated value is submitter-controlled, so escape it before it
	// reaches the support inbox as HTML.
	htmlBody := fmt.Sprintf(`
		<h2>New Contact Form Submission</h2>
		<p><strong>From:</strong> %s (%s)</p>
		<p><strong>Submitted:</strong> %s</p>
		<h3>Message:</h3>
		<p>%s</p>
		<hr>
		<p><em>This email was sent from the Powhunter contact form.</em></p>
	`,
		html.EscapeString(req.Name),
		html.EscapeString(req.Email),
		time.Now().Format("January 2, 2006 at 3:04 PM MST"),
		strings.ReplaceAll(html.EscapeString(req.Message), "\n", "<br>"),
	)

	if err := h.mailer.Send(ctx, notify.Email{
		To:      []string{h.support},
		ReplyTo: req.Email,
		Subject: fmt.Sprintf("Contact Form Submission from %s", req.Name),
		HTML:    htmlBody,
	}); err != nil {
		return fmt.Errorf("failed to send contact email: %w", err)
	}

	return nil
}
