package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"
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
	// Get SMTP configuration from environment
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASSWORD")
	// If SMTP is not configured, skip sending email
	if smtpHost == "" || smtpPort == "" {
		log.Println("SMTP not configured, skipping email send")
		return nil
	}

	toEmail := "support@powhunter.app"

	// Construct email
	subject := fmt.Sprintf("Contact Form Submission from %s", req.Name)
	body := fmt.Sprintf(`New contact form submission:

From: %s <%s>
Submitted: %s

Message:
%s

---
This email was sent from the Powhunter contact form.
`, req.Name, req.Email, time.Now().Format("January 2, 2006 at 3:04 PM MST"), req.Message)

	message := fmt.Sprintf("From: %s\r\n", req.Email)
	message += fmt.Sprintf("To: %s\r\n", toEmail)
	message += fmt.Sprintf("Reply-To: %s\r\n", req.Email)
	message += fmt.Sprintf("Subject: %s\r\n", subject)
	message += "MIME-Version: 1.0\r\n"
	message += "Content-Type: text/plain; charset=UTF-8\r\n"
	message += "\r\n"
	message += body

	// Send email
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	addr := fmt.Sprintf("%s:%s", smtpHost, smtpPort)

	err := smtp.SendMail(addr, auth, req.Email, []string{toEmail}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Contact email sent to %s from %s (%s)", toEmail, req.Name, req.Email)
	return nil
}
