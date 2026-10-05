// Package seed loads the bundled resort catalog into the database.
//
// The catalog is embedded so any binary can apply it without the source tree
// on disk; cmd/migrate runs it on every deploy so new resorts go live on merge.
package seed

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"

	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"

	"github.com/google/uuid"
)

//go:embed resorts.json
var catalog []byte

// Resort is one entry in the resort catalog.
type Resort struct {
	Name string `json:"name"`
	URL  struct {
		Host     string `json:"host"`
		PathName string `json:"pathname"`
	} `json:"url"`
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Upserter is the subset of the generated queries that seeding needs.
type Upserter interface {
	UpsertResort(ctx context.Context, arg dbgen.UpsertResortParams) (dbgen.Resort, error)
}

// Catalog parses the embedded resort catalog.
func Catalog() ([]Resort, error) {
	var resorts []Resort
	if err := json.Unmarshal(catalog, &resorts); err != nil {
		return nil, fmt.Errorf("parsing resort catalog: %w", err)
	}

	return resorts, nil
}

// Resorts upserts every catalog resort and returns how many it applied.
//
// It upserts rather than clearing and reinserting: deleting resorts cascades to
// user_alerts, so a clear would wipe every user's alerts. That also makes it
// safe to run on every deploy.
func Resorts(ctx context.Context, q Upserter, logger *slog.Logger) (int, error) {
	resorts, err := Catalog()
	if err != nil {
		return 0, err
	}

	for _, r := range resorts {
		if _, upsertErr := q.UpsertResort(ctx, upsertParams(r)); upsertErr != nil {
			return 0, fmt.Errorf("upserting resort %s: %w", r.Name, upsertErr)
		}
	}

	logger.InfoContext(ctx, "resort catalog applied", "count", len(resorts))

	return len(resorts), nil
}

// upsertParams maps a catalog resort onto the upsert query. The UUID is only
// used when the resort is new; an existing resort keeps its UUID.
func upsertParams(r Resort) dbgen.UpsertResortParams {
	return dbgen.UpsertResortParams{
		Uuid:        uuid.New(),
		Name:        r.Name,
		UrlHost:     sql.NullString{String: r.URL.Host, Valid: r.URL.Host != ""},
		UrlPathname: sql.NullString{String: r.URL.PathName, Valid: r.URL.PathName != ""},
		Latitude:    sql.NullFloat64{Float64: r.Lat, Valid: true},
		Longitude:   sql.NullFloat64{Float64: r.Lon, Valid: true},
	}
}
