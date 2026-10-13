package handlers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/notify"
)

// LinkQueries is the slice of the generated queries the alert links use.
type LinkQueries interface {
	RecordAlertLinkClick(ctx context.Context, code string) (string, error)
	GetAlertLinkDestination(ctx context.Context, code string) (string, error)
	GetAlertLinkDetails(ctx context.Context, code string) (dbgen.GetAlertLinkDetailsRow, error)
}

// LinkHandler serves the tracked links at the end of alert SMS messages and
// the details the web app's alert page shows for them.
type LinkHandler struct {
	queries    LinkQueries
	appBaseURL string
	logger     *slog.Logger
}

// NewLinkHandler builds a LinkHandler. appBaseURL is the web app: a link lands
// on its alert page, and an unknown or broken link on its home page, so a
// visitor always ends up somewhere useful.
func NewLinkHandler(queries LinkQueries, appBaseURL string, logger *slog.Logger) *LinkHandler {
	return &LinkHandler{queries: queries, appBaseURL: appBaseURL, logger: logger}
}

// AlertLinkDetails is what the alert page shows. It describes the storm and
// never the person the alert was sent to, since anyone holding the link,
// including the friends it is shared with, can read it.
type AlertLinkDetails struct {
	ResortUUID   string  `json:"resortUuid"`
	ResortName   string  `json:"resortName"`
	SnowAmount   float64 `json:"snowAmount"`
	ForecastDate string  `json:"forecastDate"`
	ReportURL    string  `json:"reportUrl"`
}

// isPreviewFetch reports whether a request should be redirected without being
// counted as a click.
//
// Messaging apps fetch a link to draw its preview, and Apple's previewer
// announces itself as facebookexternalhit. Counting those would report every
// alert as clicked.
func isPreviewFetch(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return true
	}

	agent := strings.ToLower(r.UserAgent())
	if agent == "" {
		return true
	}

	for _, hint := range []string{"bot", "facebookexternalhit", "preview", "crawler", "spider", "whatsapp"} {
		if strings.Contains(agent, hint) {
			return true
		}
	}

	return false
}

// Redirect counts a click on an alert link and sends the visitor to the web
// app's page for that alert, which offers the mountain report and a way to
// share the storm. The target is built from configuration and a code that has
// passed IsLinkCode, never from free text in the request, so this cannot be
// used as an open redirect.
func (h *LinkHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if !notify.IsLinkCode(code) {
		http.Redirect(w, r, h.appBaseURL, http.StatusFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), handlerTimeout)
	defer cancel()

	lookup := h.queries.RecordAlertLinkClick
	if isPreviewFetch(r) {
		lookup = h.queries.GetAlertLinkDestination
	}

	if _, err := lookup(ctx, code); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			h.logger.ErrorContext(ctx, "failed to resolve alert link", "error", err)
		}

		http.Redirect(w, r, h.appBaseURL, http.StatusFound)

		return
	}

	http.Redirect(w, r, h.appBaseURL+"/alert/"+code, http.StatusFound)
}

// Details returns what the alert page shows for a link. It does not count a
// click: the redirect into the page already did.
func (h *LinkHandler) Details(w http.ResponseWriter, r *http.Request) {
	setSecurityHeaders(w)

	code := r.PathValue("code")
	if !notify.IsLinkCode(code) {
		sendErrorResponse(w, "ALERT_LINK_NOT_FOUND", "That alert link does not exist", http.StatusNotFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), handlerTimeout)
	defer cancel()

	row, err := h.queries.GetAlertLinkDetails(ctx, code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			sendErrorResponse(w, "ALERT_LINK_NOT_FOUND", "That alert link does not exist", http.StatusNotFound)
			return
		}

		h.logger.ErrorContext(ctx, "failed to load alert link", "error", err)
		sendErrorResponse(w, "INTERNAL_ERROR", "Could not load that alert", http.StatusInternalServerError)

		return
	}

	writeJSON(w, http.StatusOK, AlertLinkDetails{
		ResortUUID: row.ResortUuid.String(),
		ResortName: row.ResortName,
		SnowAmount: row.SnowAmount,
		// A calendar date, not an instant: formatting it as a timestamp would
		// shift it a day for anyone west of UTC.
		ForecastDate: row.ForecastDate.Format(time.DateOnly),
		ReportURL:    row.DestinationUrl,
	})
}
