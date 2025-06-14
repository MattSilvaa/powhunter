package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/MattSilvaa/powhunter/internal/db"
)

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
	Resort *ResortHandler
	Alert  *AlertHandler
	store  *db.Store
}

// Store returns the store used by the handlers
func (h *Handlers) Store() *db.Store {
	return h.store
}

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

func NewResortHandler(store db.StoreService) (*ResortHandler, error) {
	return &ResortHandler{
		store: store,
	}, nil
}

func (h *ResortHandler) ListAllResorts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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
	Email            string   `json:"email"`
	Phone            string   `json:"phone"`
	NotificationDays int      `json:"notificationDays"`
	MinSnowAmount    int      `json:"minSnowAmount"`
	ResortsUuids     []string `json:"resortsUuids"`
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

	if req.Phone == "" || len(req.ResortsUuids) == 0 {
		http.Error(w, "Phone and at least one resort are required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	err := h.store.CreateUserWithAlerts(
		ctx,
		req.Email,
		req.Phone,
		int32(req.MinSnowAmount),
		int32(req.NotificationDays),
		req.ResortsUuids,
	)
	if err != nil {
		log.Printf("Failed to create alert: %v", err)
		http.Error(w, "Failed to create alert", http.StatusInternalServerError)
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
