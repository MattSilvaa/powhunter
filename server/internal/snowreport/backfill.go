package snowreport

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/weather"
)

// BackfillStore is what Backfill reads and writes. *dbgen.Queries satisfies it.
type BackfillStore interface {
	ListResortsMissingSeasonStats(ctx context.Context, season string) ([]dbgen.Resort, error)
	UpsertSeasonStats(ctx context.Context, arg dbgen.UpsertSeasonStatsParams) error
}

// Backfill computes last season's stats for every resort that does not have
// them yet. Once a season is stored it is never fetched again, so in practice
// this calls the archive once per resort per year. Resorts are fetched one at
// a time to stay well inside Open-Meteo's rate limits, and a failure for one
// resort does not stop the others; the next run retries whatever is missing.
func Backfill(
	ctx context.Context,
	store BackfillStore,
	archive weather.ArchiveService,
	now time.Time,
	logger *slog.Logger,
) error {
	season := LastCompletedSeason(now)

	resorts, err := store.ListResortsMissingSeasonStats(ctx, season.Name)
	if err != nil {
		return fmt.Errorf("listing resorts missing %s stats: %w", season.Name, err)
	}

	var failures []error

	for _, resort := range resorts {
		if ctx.Err() != nil {
			return errors.Join(append(failures, ctx.Err())...)
		}

		if !resort.Latitude.Valid || !resort.Longitude.Valid {
			continue
		}

		days, fetchErr := archive.GetSeasonSnowfall(ctx,
			resort.Latitude.Float64, resort.Longitude.Float64, season.Start, season.End)
		if fetchErr != nil {
			failures = append(failures, fmt.Errorf("%s: %w", resort.Name, fetchErr))

			continue
		}

		if len(days) == 0 {
			failures = append(failures, fmt.Errorf("%s: archive returned no days", resort.Name))

			continue
		}

		stats := ComputeSeasonStats(days)

		if saveErr := store.UpsertSeasonStats(ctx, dbgen.UpsertSeasonStatsParams{
			ResortUuid:        resort.Uuid,
			Season:            season.Name,
			SeasonStart:       season.Start,
			SeasonEnd:         season.End,
			TotalSnowfallIn:   stats.TotalIn,
			Days6inPlus:       int32(stats.DeepDays), //nolint:gosec // at most 181 days in a season
			BiggestStormIn:    stats.BiggestStormIn,
			BiggestStormStart: sql.NullTime{Time: stats.StormStart, Valid: !stats.StormStart.IsZero()},
			BiggestStormEnd:   sql.NullTime{Time: stats.StormEnd, Valid: !stats.StormEnd.IsZero()},
		}); saveErr != nil {
			failures = append(failures, fmt.Errorf("%s: saving stats: %w", resort.Name, saveErr))

			continue
		}

		logger.InfoContext(ctx, "stored season recap",
			"resort", resort.Name, "season", season.Name,
			"total_in", stats.TotalIn, "deep_days", stats.DeepDays, "biggest_storm_in", stats.BiggestStormIn)
	}

	return errors.Join(failures...)
}
