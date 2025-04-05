package handlers

import (
	"embed"
	"encoding/json"
	"errors"
	"net/http"
)

type Resort struct {
	Name string `json:"name"`
	URL  struct {
		Host     string `json:"host"`
		PathName string `json:"pathname"`
	} `json:"url"`
}

type ResortHandler struct {
	resorts []Resort
}

type Handlers struct {
	Resort *ResortHandler
	Alert  *AlertHandler
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
	resortHandler, err := NewResortHandler()
	if err != nil {
		return nil, err
	}

	alertHandler, err := NewAlertHandler()
	if err != nil {
		return nil, err
	}

	return &Handlers{
		Resort: resortHandler,
		Alert:  alertHandler,
	}, nil
}

func NewResortHandler() (*ResortHandler, error) {
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
	}, nil
}

func (h *ResortHandler) GetAllResorts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	setSecurityHeaders(w)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(h.resorts); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

type AlertHandler struct{}

func NewAlertHandler() (*AlertHandler, error) {
	return &AlertHandler{}, nil
}

func (h *AlertHandler) CreateAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	setSecurityHeaders(w)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("What up!"))
}
