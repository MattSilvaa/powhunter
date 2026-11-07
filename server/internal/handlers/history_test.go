//nolint:testpackage // exercises unexported handler internals, as the sibling handler tests do
package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/snowhistory"
	"github.com/MattSilvaa/powhunter/internal/weather"
)

type fakeResortLookup struct {
	resort dbgen.Resort
	err    error
}

func (f fakeResortLookup) GetResortByUUID(context.Context, uuid.UUID) (dbgen.Resort, error) {
	return f.resort, f.err
}

type fakeSnowHistory struct {
	gotThreshold float64
	err          error
}

func (f *fakeSnowHistory) Summary(_ context.Context, _ string, _, _, threshold float64) (snowhistory.Summary, error) {
	f.gotThreshold = threshold
	return snowhistory.Summary{AverageSeasonInches: 312.5}, f.err
}

func TestGetSnowHistory(t *testing.T) {
	located := dbgen.Resort{
		Name:      "Vail",
		Latitude:  sql.NullFloat64{Float64: 39.6, Valid: true},
		Longitude: sql.NullFloat64{Float64: -106.4, Valid: true},
	}

	tests := []struct {
		name          string
		resortUUID    string
		query         string
		resorts       fakeResortLookup
		historyErr    error
		wantStatus    int
		wantErrorCode string
	}{
		{
			name:          "bad uuid",
			resortUUID:    "not-a-uuid",
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "INVALID_REQUEST",
		},
		{
			name:          "threshold out of range",
			resortUUID:    testResortUUID1,
			query:         "threshold=500",
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "VALIDATION_ERROR",
		},
		{
			name:          "unknown resort",
			resortUUID:    testResortUUID1,
			resorts:       fakeResortLookup{err: sql.ErrNoRows},
			wantStatus:    http.StatusNotFound,
			wantErrorCode: "RESORT_NOT_FOUND",
		},
		{
			name:          "resort without coordinates",
			resortUUID:    testResortUUID1,
			resorts:       fakeResortLookup{resort: dbgen.Resort{Name: "Nowhere"}},
			wantStatus:    http.StatusNotFound,
			wantErrorCode: "RESORT_NOT_FOUND",
		},
		{
			name:          "rate limited upstream",
			resortUUID:    testResortUUID1,
			resorts:       fakeResortLookup{resort: located},
			historyErr:    fmt.Errorf("fetching: %w", &weather.RateLimitError{}),
			wantStatus:    http.StatusServiceUnavailable,
			wantErrorCode: "HISTORY_UNAVAILABLE",
		},
		{
			name:          "other upstream failure",
			resortUUID:    testResortUUID1,
			resorts:       fakeResortLookup{resort: located},
			historyErr:    errors.New("boom"),
			wantStatus:    http.StatusBadGateway,
			wantErrorCode: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHistoryHandler(tt.resorts, &fakeSnowHistory{err: tt.historyErr},
				slog.New(slog.NewTextHandler(io.Discard, nil)))

			rec := serveSnowHistory(handler, tt.resortUUID, tt.query)

			assert.Equal(t, tt.wantStatus, rec.Code)

			var body ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			assert.Equal(t, tt.wantErrorCode, body.Error)
		})
	}
}

func TestGetSnowHistorySuccess(t *testing.T) {
	history := &fakeSnowHistory{}
	handler := NewHistoryHandler(fakeResortLookup{resort: dbgen.Resort{
		Name:      "Vail",
		Latitude:  sql.NullFloat64{Float64: 39.6, Valid: true},
		Longitude: sql.NullFloat64{Float64: -106.4, Valid: true},
	}}, history, slog.New(slog.NewTextHandler(io.Discard, nil)))

	rec := serveSnowHistory(handler, testResortUUID1, "threshold=6")

	require.Equal(t, http.StatusOK, rec.Code)
	assert.InDelta(t, 6.0, history.gotThreshold, 0.001)

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "Vail", body["resortName"])
	assert.Equal(t, testResortUUID1, body["resortUuid"])
	assert.InDelta(t, 6.0, body["threshold"], 0.001)
	assert.InDelta(t, 312.5, body["averageSeasonInches"], 0.001)
}

// serveSnowHistory routes through a mux so the {uuid} path value is set.
func serveSnowHistory(handler *HistoryHandler, resortUUID, query string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/resorts/{uuid}/snow-history", handler.GetSnowHistory)

	target := "/api/resorts/" + resortUUID + "/snow-history?" + query
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))

	return rec
}
