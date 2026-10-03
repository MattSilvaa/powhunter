package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/MattSilvaa/powhunter/internal/db"
	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"

	"github.com/google/uuid"
)

const resortsPath = "cmd/seed/data/resorts.json"

// Resort represents a ski resort from the JSON file.
type Resort struct {
	Name string `json:"name"`
	URL  struct {
		Host     string `json:"host"`
		PathName string `json:"pathname"`
	} `json:"url"`
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// loadResorts reads and parses the resort seed file.
func loadResorts(path string) ([]Resort, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading resorts data: %w", err)
	}

	var resorts []Resort
	if err := json.Unmarshal(data, &resorts); err != nil {
		return nil, fmt.Errorf("parsing resorts data: %w", err)
	}

	return resorts, nil
}

// upsertParams maps a seed resort onto the upsert query. The UUID is only
// used when the resort is new; an existing resort keeps its UUID.
func upsertParams(r Resort) dbgen.UpsertResortParams {
	return dbgen.UpsertResortParams{
		Uuid: uuid.New(),
		Name: r.Name,
		UrlHost: sql.NullString{
			String: r.URL.Host,
			Valid:  r.URL.Host != "",
		},
		UrlPathname: sql.NullString{
			String: r.URL.PathName,
			Valid:  r.URL.PathName != "",
		},
		Latitude: sql.NullFloat64{
			Float64: r.Lat,
			Valid:   true,
		},
		Longitude: sql.NullFloat64{
			Float64: r.Lon,
			Valid:   true,
		},
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("resort seeding failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	resorts, err := loadResorts(resortsPath)
	if err != nil {
		return err
	}

	dbConn, err := db.New()
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer dbConn.Close()

	queries := dbgen.New(dbConn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Upsert rather than clear-and-insert: deleting resorts cascades to
	// user_alerts, so re-seeding would wipe every user's alerts.
	for _, r := range resorts {
		if _, upsertErr := queries.UpsertResort(ctx, upsertParams(r)); upsertErr != nil {
			return fmt.Errorf("upserting resort %s: %w", r.Name, upsertErr)
		}
		logger.Info("upserted resort", "name", r.Name)
	}

	logger.Info("resort seeding completed", "count", len(resorts))

	return nil
}
