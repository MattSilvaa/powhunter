package handlers

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/MattSilvaa/powhunter/internal/db"
	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
)

type Resort struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
	URL  struct {
		Host     string `json:"host"`
		PathName string `json:"pathname"`
	} `json:"url"`
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Noaa string  `json:"noaa"`
}

type ResortHandler struct {
	resorts []Resort
	store   *db.Store
}

type AlertHandler struct {
	store *db.Store
}

type Handlers struct {
	Resort *ResortHandler
	Alert  *AlertHandler
	store  *db.Store
}

//go:embed "data/resorts.json"
var resortsFS embed.FS

func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
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

	return &Handlers{
		Resort: resortHandler,
		Alert:  alertHandler,
		store:  store,
	}, nil
}

func NewResortHandler(store *db.Store) (*ResortHandler, error) {
	// For now, keep using the embedded JSON file
	// In the future, you'll want to migrate this data to the database
	data, err := resortsFS.ReadFile("data/resorts.json")
	if err != nil {
		return nil, errors.New("failed to read resorts data: " + err.Error())
	}

	var resorts []Resort
	if err := json.Unmarshal(data, &resorts); err != nil {
		return nil, errors.New("failed to parse resorts data: " + err.Error())
	}

	return &ResortHandler{
		resorts: resorts,
		store:   store,
	}, nil
}

func (h *ResortHandler) GetAllResorts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	setSecurityHeaders(w)

	// In the future, fetch from database instead
	// resorts, err := h.store.Queries.ListResorts(r.Context())

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(h.resorts); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

type CreateAlertRequest struct {
	Email            string   `json:"email"`
	Phone            string   `json:"phone"`
	NotificationDays int      `json:"notificationDays"`
	MinSnowAmount    int      `json:"minSnowAmount"`
	Resorts          []string `json:"resorts"`
}

func NewAlertHandler(store *db.Store) (*AlertHandler, error) {
	return &AlertHandler{
		store: store,
	}, nil
}

func (h *AlertHandler) CreateAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	setSecurityHeaders(w)

	var req CreateAlertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Input validation
	if req.Email == "" || len(req.Resorts) == 0 {
		http.Error(w, "Email and at least one resort are required", http.StatusBadRequest)
		return
	}

	// Set a timeout for the database operation
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Create user and alerts in the database
	err := h.store.CreateUserWithAlerts(
		ctx,
		req.Email,
		req.Phone,
		int32(req.MinSnowAmount),
		int32(req.NotificationDays),
		req.Resorts,
	)

	if err != nil {
		log.Printf("Failed to create alert: %v", err)

		// Check for duplicate email
		if err.Error() == "error creating user: ERROR: duplicate key value violates unique constraint \"users_email_key\"" {
			http.Error(w, "User with this email already exists", http.StatusConflict)
			return
		}

		http.Error(w, "Failed to create alert", http.StatusInternalServerError)
		return
	}

	// Respond with success
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Alert created successfully",
	})
}
