package handlers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"github.com/MattSilvaa/powhunter/internal/db"
	"github.com/MattSilvaa/powhunter/internal/middleware"
	"github.com/MattSilvaa/powhunter/internal/validate"
	"github.com/google/uuid"
)

// AccountHandler serves the signed-in user's changes to their own account:
// adding alerts, editing them, and changing the phone alerts go to. Every
// endpoint takes the account from the session, never from the request.
type AccountHandler struct {
	store  db.StoreService
	logger *slog.Logger
}

// NewAccountHandler builds the account endpoints.
func NewAccountHandler(store db.StoreService, logger *slog.Logger) *AccountHandler {
	return &AccountHandler{store: store, logger: logger}
}

// AddAlertsRequest adds resorts to a signed-in account. It carries no email or
// phone: those belong to the session's account.
type AddAlertsRequest struct {
	NotificationDays int32    `json:"notificationDays"`
	MinSnowAmount    float64  `json:"minSnowAmount"`
	ResortsUuids     []string `json:"resortsUuids"`
}

// UpdateAlertRequest changes the thresholds on one alert.
type UpdateAlertRequest struct {
	NotificationDays int32   `json:"notificationDays"`
	MinSnowAmount    float64 `json:"minSnowAmount"`
}

// UpdateProfileRequest changes the account's contact details.
type UpdateProfileRequest struct {
	Phone string `json:"phone"`
}

// CreateUserAlerts adds alerts for more resorts to the signed-in account.
func (h *AccountHandler) CreateUserAlerts(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		sendErrorResponse(w, "UNAUTHENTICATED", "Please sign in to continue", http.StatusUnauthorized)
		return
	}

	var req AddAlertsRequest
	if err := decodeJSON(r, &req); err != nil {
		sendDecodeError(w, err)
		return
	}

	if !validateResorts(w, req.ResortsUuids) ||
		!validateAlertSettings(w, int(req.NotificationDays), req.MinSnowAmount) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), handlerTimeout)
	defer cancel()

	err := h.store.CreateAlertsForUser(
		ctx,
		user.UUID,
		req.MinSnowAmount,
		req.NotificationDays,
		req.ResortsUuids,
	)
	if err != nil {
		h.logError(r, "failed to add alerts", err)
		sendAlertWriteError(w, err)

		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"status":  "success",
		"message": "Alerts created successfully",
	})
}

// UpdateUserAlert changes the thresholds on one of the signed-in user's alerts.
func (h *AccountHandler) UpdateUserAlert(w http.ResponseWriter, r *http.Request) {
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

	var req UpdateAlertRequest
	if err := decodeJSON(r, &req); err != nil {
		sendDecodeError(w, err)
		return
	}

	if !validateAlertSettings(w, int(req.NotificationDays), req.MinSnowAmount) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), handlerTimeout)
	defer cancel()

	alert, err := h.store.UpdateAlertForUser(
		ctx,
		user.UUID,
		resortUUID,
		req.MinSnowAmount,
		req.NotificationDays,
	)
	if err != nil {
		// Another account's alert and a missing one look the same: the update is
		// scoped to the session's user, so neither matches a row.
		if errors.Is(err, sql.ErrNoRows) {
			sendErrorResponse(w, "ALERT_NOT_FOUND", "That alert no longer exists", http.StatusNotFound)
			return
		}

		h.logError(r, "failed to update alert", err)
		sendErrorResponse(w, "INTERNAL_ERROR", "Failed to update alert", http.StatusInternalServerError)

		return
	}

	writeJSON(w, http.StatusOK, alert)
}

// UpdateProfile changes the phone number the signed-in user's alerts go to.
func (h *AccountHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.UserFromContext(r.Context())
	if !ok {
		sendErrorResponse(w, "UNAUTHENTICATED", "Please sign in to continue", http.StatusUnauthorized)
		return
	}

	var req UpdateProfileRequest
	if err := decodeJSON(r, &req); err != nil {
		sendDecodeError(w, err)
		return
	}

	phone, err := validate.Phone(req.Phone)
	if err != nil {
		sendErrorResponse(w, "MISSING_PHONE", "Please enter a valid phone number", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), handlerTimeout)
	defer cancel()

	if updateErr := h.store.UpdateUserPhone(ctx, user.UUID, phone); updateErr != nil {
		h.logError(r, "failed to update phone", updateErr)
		sendErrorResponse(w, "INTERNAL_ERROR", "Failed to update your details", http.StatusInternalServerError)

		return
	}

	writeJSON(w, http.StatusOK, userResponse{
		Email:         user.Email,
		Phone:         phone,
		EmailVerified: user.EmailVerified,
	})
}

func (h *AccountHandler) logError(r *http.Request, msg string, err error) {
	h.logger.ErrorContext(r.Context(), msg,
		"error", err,
		"request_id", middleware.RequestIDFromContext(r.Context()),
	)
}
