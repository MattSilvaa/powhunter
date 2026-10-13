package handlers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/snowhistory"
	"github.com/MattSilvaa/powhunter/internal/validate"
	"github.com/MattSilvaa/powhunter/internal/weather"
	"github.com/google/uuid"
)

// historyTimeout is longer than handlerTimeout because a resort's first view of
// the day waits on Open-Meteo's archive, not just the database.
const historyTimeout = 25 * time.Second

// ResortLookup is the slice of the generated queries the history page uses.
type ResortLookup interface {
	GetResortByUUID(ctx context.Context, resortUUID uuid.UUID) (dbgen.Resort, error)
}

// SnowHistory supplies a resort's snowfall summary.
type SnowHistory interface {
	Summary(ctx context.Context, key string, lat, lon, threshold float64) (snowhistory.Summary, error)
}

// HistoryHandler serves a resort's past snowfall.
type HistoryHandler struct {
	resorts ResortLookup
	history SnowHistory
	logger  *slog.Logger
}

// NewHistoryHandler builds a HistoryHandler.
func NewHistoryHandler(resorts ResortLookup, history SnowHistory, logger *slog.Logger) *HistoryHandler {
	return &HistoryHandler{resorts: resorts, history: history, logger: logger}
}

// SnowHistoryResponse is a resort's snowfall history.
type SnowHistoryResponse struct {
	ResortUUID string   `json:"resortUuid"`
	ResortName string   `json:"resortName"`
	Threshold  *float64 `json:"threshold"`
	snowhistory.Summary
}

// GetSnowHistory returns a resort's snowfall over recent seasons. An optional
// threshold query parameter adds how many days each season reached it, which is
// what tells someone whether their alert threshold is realistic there.
//
// It is public: the data is the same for everyone, and the cache behind it
// bounds what repeated requests cost upstream.
func (h *HistoryHandler) GetSnowHistory(w http.ResponseWriter, r *http.Request) {
	setSecurityHeaders(w)

	resortUUID, err := uuid.Parse(r.PathValue("uuid"))
	if err != nil {
		sendErrorResponse(w, "INVALID_REQUEST", "Invalid resort", http.StatusBadRequest)
		return
	}

	var threshold *float64
	if raw := r.URL.Query().Get("threshold"); raw != "" {
		value, parseErr := strconv.ParseFloat(raw, 64)
		if parseErr != nil || validate.SnowAmount(value) != nil {
			sendErrorResponse(w, "VALIDATION_ERROR", "Invalid snow threshold", http.StatusBadRequest)
			return
		}

		threshold = &value
	}

	ctx, cancel := context.WithTimeout(r.Context(), historyTimeout)
	defer cancel()

	resort, err := h.resorts.GetResortByUUID(ctx, resortUUID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && (!resort.Latitude.Valid || !resort.Longitude.Valid)) {
		sendErrorResponse(w, "RESORT_NOT_FOUND", "Resort not found", http.StatusNotFound)
		return
	}

	if err != nil {
		h.logger.ErrorContext(ctx, "failed to look up resort", "resort_uuid", resortUUID, "error", err)
		sendErrorResponse(w, "INTERNAL_ERROR", "Failed to load snowfall history", http.StatusInternalServerError)

		return
	}

	var minInches float64
	if threshold != nil {
		minInches = *threshold
	}

	summary, err := h.history.Summary(
		ctx,
		resortUUID.String(),
		resort.Latitude.Float64,
		resort.Longitude.Float64,
		minInches,
	)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to load snowfall history", "resort_uuid", resortUUID, "error", err)

		if errors.Is(err, weather.ErrRateLimited) {
			sendErrorResponse(w, "HISTORY_UNAVAILABLE", "Snowfall history is busy, try again shortly",
				http.StatusServiceUnavailable)

			return
		}

		sendErrorResponse(w, "INTERNAL_ERROR", "Failed to load snowfall history", http.StatusBadGateway)

		return
	}

	writeJSON(w, http.StatusOK, SnowHistoryResponse{
		ResortUUID: resortUUID.String(),
		ResortName: resort.Name,
		Threshold:  threshold,
		Summary:    summary,
	})
}
