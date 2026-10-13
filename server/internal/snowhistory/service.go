package snowhistory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/MattSilvaa/powhunter/internal/weather"
)

const (
	// cacheTTL is how long a resort's history is reused. Past snowfall does not
	// change, so this only bounds how stale the current season's total gets.
	cacheTTL = 24 * time.Hour

	// fetchTimeout bounds one archive request. It is detached from the caller's
	// context because other requests may be waiting on the same fetch.
	fetchTimeout = 20 * time.Second
)

// Source supplies observed daily snowfall.
type Source interface {
	GetDailySnowfall(ctx context.Context, lat, lon float64, start, end time.Time) ([]weather.DailySnowfall, error)
}

type cacheEntry struct {
	days      []weather.DailySnowfall
	fetchedAt time.Time
}

// Service serves resort snowfall summaries, caching the underlying archive
// data in memory. Open-Meteo rate limits by IP, and the forecaster shares that
// quota, so each resort is fetched at most once a day however often its
// history page is viewed.
type Service struct {
	source Source
	now    func() time.Time

	mu    sync.Mutex
	cache map[string]cacheEntry
	group singleflight.Group
}

// NewService builds a Service backed by source.
func NewService(source Source) *Service {
	return &Service{
		source: source,
		now:    time.Now,
		cache:  make(map[string]cacheEntry),
	}
}

// Summary returns the snowfall history for the resort identified by key at the
// given coordinates. A threshold of zero leaves the day counts out.
func (s *Service) Summary(ctx context.Context, key string, lat, lon, threshold float64) (Summary, error) {
	now := s.now().UTC()

	days, err := s.days(ctx, key, lat, lon, now)
	if err != nil {
		return Summary{}, err
	}

	return Summarize(days, threshold, now), nil
}

func (s *Service) days(
	ctx context.Context,
	key string,
	lat, lon float64,
	now time.Time,
) ([]weather.DailySnowfall, error) {
	s.mu.Lock()
	entry, ok := s.cache[key]
	s.mu.Unlock()

	if ok && now.Sub(entry.fetchedAt) < cacheTTL {
		return entry.days, nil
	}

	result, err, _ := s.group.Do(key, func() (any, error) {
		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), fetchTimeout)
		defer cancel()

		start, end := Window(now)
		days, err := s.source.GetDailySnowfall(fetchCtx, lat, lon, start, end)
		if err != nil {
			return nil, fmt.Errorf("fetching snowfall history: %w", err)
		}

		s.mu.Lock()
		s.cache[key] = cacheEntry{days: days, fetchedAt: now}
		s.mu.Unlock()

		return days, nil
	})
	if err != nil {
		return nil, err
	}

	days, ok := result.([]weather.DailySnowfall)
	if !ok {
		return nil, fmt.Errorf("unexpected snowfall history type %T", result)
	}

	return days, nil
}
