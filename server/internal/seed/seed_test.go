package seed_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/seed"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The catalog is hand-edited, so guard against typos such as swapped or
// missing coordinates and duplicate names (which the upsert would silently merge).
func TestCatalog(t *testing.T) {
	resorts, err := seed.Catalog()
	require.NoError(t, err)
	require.NotEmpty(t, resorts)

	seen := map[string]bool{}
	for _, r := range resorts {
		assert.NotEmpty(t, r.Name)
		assert.False(t, seen[r.Name], "duplicate resort %q", r.Name)
		seen[r.Name] = true

		assert.NotEmpty(t, r.URL.Host, r.Name)
		assert.NotEmpty(t, r.URL.PathName, r.Name)

		// Rough North America bounds.
		assert.True(t, r.Lat > 24 && r.Lat < 70, "%s latitude %v", r.Name, r.Lat)
		assert.True(t, r.Lon > -170 && r.Lon < -50, "%s longitude %v", r.Name, r.Lon)
	}
}

type fakeUpserter struct {
	got []dbgen.UpsertResortParams
	err error
}

func (f *fakeUpserter) UpsertResort(_ context.Context, arg dbgen.UpsertResortParams) (dbgen.Resort, error) {
	f.got = append(f.got, arg)
	return dbgen.Resort{}, f.err
}

func TestResortsUpsertsWholeCatalog(t *testing.T) {
	resorts, err := seed.Catalog()
	require.NoError(t, err)

	fake := &fakeUpserter{}
	n, err := seed.Resorts(t.Context(), fake, slog.New(slog.NewTextHandler(io.Discard, nil)))

	require.NoError(t, err)
	assert.Equal(t, len(resorts), n)
	require.Len(t, fake.got, len(resorts))
	assert.Equal(t, resorts[0].Name, fake.got[0].Name)
	assert.InDelta(t, resorts[0].Lat, fake.got[0].Latitude.Float64, 1e-9)
}

func TestResortsStopsOnError(t *testing.T) {
	fake := &fakeUpserter{err: errors.New("boom")}
	_, err := seed.Resorts(t.Context(), fake, slog.New(slog.NewTextHandler(io.Discard, nil)))

	require.Error(t, err)
	assert.Len(t, fake.got, 1)
}
