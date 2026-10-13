package handlers

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/MattSilvaa/powhunter/internal/notify"
)

// LinkQueries is the slice of the generated queries the link redirect uses.
type LinkQueries interface {
	RecordAlertLinkClick(ctx context.Context, code string) (string, error)
	GetAlertLinkDestination(ctx context.Context, code string) (string, error)
}

// LinkHandler serves the tracked links at the end of alert SMS messages.
type LinkHandler struct {
	queries  LinkQueries
	fallback string
	logger   *slog.Logger
}

// NewLinkHandler builds a LinkHandler. Fallback is where an unknown or broken
// link lands, so a visitor always ends up somewhere useful.
func NewLinkHandler(queries LinkQueries, fallback string, logger *slog.Logger) *LinkHandler {
	return &LinkHandler{queries: queries, fallback: fallback, logger: logger}
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

// Redirect counts a click on an alert link and sends the visitor on to the
// link's destination. The destination comes only from the database, never from
// the request, so this cannot be used as an open redirect.
func (h *LinkHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if !notify.IsLinkCode(code) {
		http.Redirect(w, r, h.fallback, http.StatusFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), handlerTimeout)
	defer cancel()

	lookup := h.queries.RecordAlertLinkClick
	if isPreviewFetch(r) {
		lookup = h.queries.GetAlertLinkDestination
	}

	destination, err := lookup(ctx, code)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			h.logger.ErrorContext(ctx, "failed to resolve alert link", "error", err)
		}

		http.Redirect(w, r, h.fallback, http.StatusFound)

		return
	}

	http.Redirect(w, r, destination, http.StatusFound)
}
