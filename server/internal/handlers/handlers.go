package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/MattSilvaa/powhunter/internal/auth"
	"github.com/MattSilvaa/powhunter/internal/db"
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
	store   *db.Store
}

var METHOD_NOT_ALLOWED = "METHOD_NOT_ALLOWED"

// Store returns the store used by the handlers.
func (h *Handlers) Store() *db.Store {
	return h.store
}

func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
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

func NewHandlers() (*Handlers, error) {
	dbConn, err := db.New()
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

	contactHandler, err := NewContactHandler()
	if err != nil {
		return nil, err
	}

	return &Handlers{
		Resort:  resortHandler,
		Alert:   alertHandler,
		Contact: contactHandler,
		store:   store,
	}, nil
}

func NewResortHandler(store db.StoreService) (*ResortHandler, error) {
	return &ResortHandler{
		store: store,
	}, nil
}

func (h *ResortHandler) ListAllResorts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendErrorResponse(w, METHOD_NOT_ALLOWED, "Method not allowed", http.StatusMethodNotAllowed)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	setSecurityHeaders(w)

	resorts, err := h.store.ListAllResorts(ctx)
	if err != nil {
		http.Error(w, "Failed to retrieve resorts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resorts); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func NewAlertHandler(store db.StoreService) (*AlertHandler, error) {
	return &AlertHandler{
		store: store,
	}, nil
}

type CreateAlertRequest struct {
	Phone            string   `json:"phone"`
	NotificationDays int      `json:"notificationDays"`
	MinSnowAmount    float64  `json:"minSnowAmount"`
	ResortsUuids     []string `json:"resortsUuids"`
}

func (h *AlertHandler) GetUserAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendErrorResponse(w, "METHOD_NOT_ALLOWED", "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	setSecurityHeaders(w)

	// Get authenticated user from context
	claims := r.Context().Value("claims").(*auth.JWTClaims)
	if claims == nil {
		sendErrorResponse(w, "UNAUTHORIZED", "Authentication required", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	alerts, err := h.store.GetUserAlertsByEmail(ctx, claims.Email)
	if err != nil {
		log.Printf("Failed to get user alerts: %v", err)
		sendErrorResponse(w, "INTERNAL_ERROR", "Failed to retrieve alerts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(alerts); err != nil {
		log.Printf("Failed to encode alerts response: %v", err)
		sendErrorResponse(w, "INTERNAL_ERROR", "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *AlertHandler) DeleteUserAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		sendErrorResponse(w, "METHOD_NOT_ALLOWED", "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	setSecurityHeaders(w)

	// Get authenticated user from context
	claims := r.Context().Value("claims").(*auth.JWTClaims)
	if claims == nil {
		sendErrorResponse(w, "UNAUTHORIZED", "Authentication required", http.StatusUnauthorized)
		return
	}

	resortUuid := r.URL.Query().Get("resort_uuid")
	if resortUuid == "" {
		sendErrorResponse(w, "MISSING_RESORT", "Resort UUID parameter is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	err := h.store.DeleteUserAlert(ctx, claims.Email, resortUuid)
	if err != nil {
		log.Printf("Failed to delete user alert: %v", err)
		sendErrorResponse(w, "INTERNAL_ERROR", "Failed to delete alert", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Alert deleted successfully",
	})
}

func (h *AlertHandler) DeleteAllUserAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		sendErrorResponse(w, "METHOD_NOT_ALLOWED", "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	setSecurityHeaders(w)

	// Get authenticated user from context
	claims := r.Context().Value("claims").(*auth.JWTClaims)
	if claims == nil {
		sendErrorResponse(w, "UNAUTHORIZED", "Authentication required", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	err := h.store.DeleteAllUserAlerts(ctx, claims.Email)
	if err != nil {
		log.Printf("Failed to delete all user alerts: %v", err)
		sendErrorResponse(w, "INTERNAL_ERROR", "Failed to delete alerts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
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

	// Get authenticated user from context
	claims := r.Context().Value("claims").(*auth.JWTClaims)
	if claims == nil {
		sendErrorResponse(w, "UNAUTHORIZED", "Authentication required", http.StatusUnauthorized)
		return
	}

	var req CreateAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, "INVALID_REQUEST", "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Phone == "" {
		sendErrorResponse(w, "MISSING_PHONE", "Phone number is required", http.StatusBadRequest)
		return
	}

	if len(req.ResortsUuids) == 0 {
		sendErrorResponse(w, "MISSING_RESORTS", "At least one resort is required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Use email from authenticated user
	err := h.store.CreateUserWithAlerts(
		ctx,
		claims.Email,
		req.Phone,
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

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Alert created successfully",
	})

	if err != nil {
		log.Printf("Failed to write resposne: %v", err)
		return
	}
}
