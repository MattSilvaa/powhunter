package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/MattSilvaa/powhunter/internal/auth"
	"github.com/MattSilvaa/powhunter/internal/config"
	"github.com/MattSilvaa/powhunter/internal/db"
	"github.com/MattSilvaa/powhunter/internal/middleware"
	"github.com/MattSilvaa/powhunter/internal/notify"
	"github.com/MattSilvaa/powhunter/internal/validate"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type Resort struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
	URL  struct {
		Host     string `json:"host"`
		PathName string `json:"pathname"`
	} `json:"url"`
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type ResortHandler struct {
	resorts []Resort
	store   db.StoreService
}

type AlertHandler struct {
	store db.StoreService
}

type Handlers struct {
	Resort  *ResortHandler
	Alert   *AlertHandler
	Contact *ContactHandler
	Auth    *AuthHandler
	auth    *auth.Service
	store   *db.Store
}

var METHOD_NOT_ALLOWED = "METHOD_NOT_ALLOWED"

// handlerTimeout bounds how long a single request may occupy a database
// connection. The pool is small, so a long timeout here starves the API.
const handlerTimeout = 10 * time.Second

// decodeJSON reads a JSON body, rejecting unknown fields so a typo in a client
// payload is reported rather than silently ignored.
func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decoding request body: %w", err)
	}

	return nil
}

// sendDecodeError distinguishes a body that exceeded the configured limit from
// one that was merely malformed.
func sendDecodeError(w http.ResponseWriter, err error) {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		sendErrorResponse(w, "REQUEST_TOO_LARGE", "Request body is too large", http.StatusRequestEntityTooLarge)
		return
	}

	sendErrorResponse(w, "INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
}

func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

// writeJSON encodes a successful response body.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

func sendErrorResponse(w http.ResponseWriter, errorCode string, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := ErrorResponse{
		Error:   errorCode,
		Message: message,
	}

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		log.Printf("Failed to encode error response: %v", err)
	}
}

func NewHandlers(cfg config.Config, logger *slog.Logger) (*Handlers, error) {
	dbConn, err := db.Open(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	store := db.NewStore(dbConn)

	resortHandler, err := NewResortHandler(store)
	if err != nil {
		return nil, err
	}

	alertHandler, err := NewAlertHandler(store)
	if err != nil {
		return nil, err
	}

	// Without an API key mail is logged rather than sent, so local development
	// works without a Resend account. config.Load makes the key mandatory in
	// production, where that fallback would strand every user at the login screen.
	var mailer notify.Mailer = notify.NewLogMailer(logger)
	if cfg.ResendAPIKey != "" {
		mailer = notify.NewResendMailer(cfg.ResendAPIKey, cfg.MailFrom)
	}

	contactHandler, err := NewContactHandler(mailer, cfg.SupportEmail)
	if err != nil {
		return nil, err
	}

	authService := auth.NewService(store.Queries(), cfg.IsProduction(), cfg.CookieCrossSite)

	return &Handlers{
		Resort:  resortHandler,
		Alert:   alertHandler,
		Contact: contactHandler,
		Auth:    NewAuthHandler(authService, mailer, cfg.AppBaseURL, logger),
		auth:    authService,
		store:   store,
	}, nil
}

// Store returns the store used by the handlers.
func (h *Handlers) Store() *db.Store {
	return h.store
}

// AuthService returns the authentication service, for the session middleware
// and the background reaper.
func (h *Handlers) AuthService() *auth.Service {
	return h.auth
}

func NewResortHandler(store db.StoreService) (*ResortHandler, error) {
	return &ResortHandler{
		store: store,
	}, nil
}

func (h *ResortHandler) ListAllResorts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendErrorResponse(w, METHOD_NOT_ALLOWED, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), handlerTimeout)
	defer cancel()

	setSecurityHeaders(w)

	resorts, err := h.store.ListAllResorts(ctx)
	if err != nil {
		log.Printf("Failed to retrieve resorts: %v", err)
		sendErrorResponse(w, "INTERNAL_ERROR", "Failed to retrieve resorts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resorts); err != nil {
		log.Printf("Failed to encode resorts response: %v", err)
		return
	}
}

func NewAlertHandler(store db.StoreService) (*AlertHandler, error) {
	return &AlertHandler{
		store: store,
	}, nil
}

type CreateAlertRequest struct {
	Email            string   `json:"email"`
	Phone            string   `json:"phone"`
	NotificationDays int      `json:"notificationDays"`
	MinSnowAmount    float64  `json:"minSnowAmount"`
	ResortsUuids     []string `json:"resortsUuids"`
}

// The three handlers below take the account from the session established by
// middleware.RequireUser. They previously read an email out of the query
// string, which meant anyone who guessed an address could read or delete that
// person's alerts.

func (h *AlertHandler) GetUserAlerts(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		sendErrorResponse(w, "UNAUTHENTICATED", "Please sign in to continue", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), handlerTimeout)
	defer cancel()

	alerts, err := h.store.GetUserAlerts(ctx, user.UUID)
	if err != nil {
		log.Printf("Failed to get user alerts: %v", err)
		sendErrorResponse(w, "INTERNAL_ERROR", "Failed to retrieve alerts", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, alerts)
}

func (h *AlertHandler) DeleteUserAlert(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		sendErrorResponse(w, "UNAUTHENTICATED", "Please sign in to continue", http.StatusUnauthorized)
		return
	}

	resortUUID := r.URL.Query().Get("resort_uuid")
	if _, err := uuid.Parse(resortUUID); err != nil {
		sendErrorResponse(w, "MISSING_RESORT", "A valid resort UUID is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), handlerTimeout)
	defer cancel()

	if err := h.store.DeleteAlertForUser(ctx, user.UUID, resortUUID); err != nil {
		log.Printf("Failed to delete user alert: %v", err)
		sendErrorResponse(w, "INTERNAL_ERROR", "Failed to delete alert", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Alert deleted successfully",
	})
}

func (h *AlertHandler) DeleteAllUserAlerts(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		sendErrorResponse(w, "UNAUTHENTICATED", "Please sign in to continue", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), handlerTimeout)
	defer cancel()

	if err := h.store.DeleteAllAlertsForUser(ctx, user.UUID); err != nil {
		log.Printf("Failed to delete all user alerts: %v", err)
		sendErrorResponse(w, "INTERNAL_ERROR", "Failed to delete alerts", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "All alerts deleted successfully",
	})
}

func (h *AlertHandler) CreateAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendErrorResponse(w, "METHOD_NOT_ALLOWED", "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	setSecurityHeaders(w)

	var req CreateAlertRequest
	if err := decodeJSON(r, &req); err != nil {
		sendDecodeError(w, err)
		return
	}

	email, err := validate.Email(req.Email)
	if err != nil {
		sendErrorResponse(w, "MISSING_EMAIL", "Please enter a valid email address", http.StatusBadRequest)
		return
	}

	phone, err := validate.Phone(req.Phone)
	if err != nil {
		sendErrorResponse(w, "MISSING_PHONE", "Please enter a valid phone number", http.StatusBadRequest)
		return
	}

	if len(req.ResortsUuids) == 0 {
		sendErrorResponse(w, "MISSING_RESORTS", "At least one resort is required", http.StatusBadRequest)
		return
	}

	if len(req.ResortsUuids) > validate.MaxResortsPerRequest {
		sendErrorResponse(w, "VALIDATION_ERROR", "Too many resorts selected", http.StatusBadRequest)
		return
	}

	if daysErr := validate.NotificationDays(req.NotificationDays); daysErr != nil {
		sendErrorResponse(w, "VALIDATION_ERROR", "Notification days must be between 1 and 10", http.StatusBadRequest)
		return
	}

	if snowErr := validate.SnowAmount(req.MinSnowAmount); snowErr != nil {
		sendErrorResponse(w, "VALIDATION_ERROR", "Snow amount must be between 0.5 and 24 inches", http.StatusBadRequest)
		return
	}

	for _, resortUUID := range req.ResortsUuids {
		if _, parseErr := uuid.Parse(resortUUID); parseErr != nil {
			sendErrorResponse(w, "VALIDATION_ERROR", "One of the selected resorts is not valid", http.StatusBadRequest)
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), handlerTimeout)
	defer cancel()

	err = h.store.CreateUserWithAlerts(
		ctx,
		email,
		phone,
		req.MinSnowAmount,
		int32(req.NotificationDays),
		req.ResortsUuids,
	)
	if err != nil {
		log.Printf("Failed to create alert: %v", err)

		pqErr := &pq.Error{}
		if errors.As(err, &pqErr) {
			switch pqErr.Code {
			case "23505": // unique_violation
				if pqErr.Constraint == "user_alerts_user_uuid_resort_uuid_key" {
					sendErrorResponse(
						w,
						"DUPLICATE_ALERT",
						"You already have an alert for this resort",
						http.StatusConflict,
					)
					return
				}
				sendErrorResponse(w, "DUPLICATE_ENTRY", "This entry already exists", http.StatusConflict)
				return
			case "23502": // not_null_violation
				sendErrorResponse(w, "MISSING_REQUIRED_FIELD", "Required field is missing", http.StatusBadRequest)
				return
			case "23514": // check_violation
				sendErrorResponse(w, "VALIDATION_ERROR", "Data validation failed", http.StatusBadRequest)
				return
			}
		}

		sendErrorResponse(w, "INTERNAL_ERROR", "Failed to create alert", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"status":  "success",
		"message": "Alert created successfully",
	})
}
