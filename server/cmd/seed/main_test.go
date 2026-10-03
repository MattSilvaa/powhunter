package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The seed file is hand-edited, so guard against typos such as swapped or
// missing coordinates and duplicate names (which the upsert would silently merge).
func TestResortsSeedFile(t *testing.T) {
	resorts, err := loadResorts("data/resorts.json")
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

func TestLoadResortsMissingFile(t *testing.T) {
	_, err := loadResorts("data/does-not-exist.json")
	assert.Error(t, err)
}

func TestUpsertParams(t *testing.T) {
	var r Resort
	r.Name = "Alta"
	r.URL.Host = "https://www.alta.com"
	r.Lat, r.Lon = 40.5884, -111.6386

	p := upsertParams(r)

	assert.Equal(t, "Alta", p.Name)
	assert.True(t, p.UrlHost.Valid)
	assert.False(t, p.UrlPathname.Valid, "empty pathname should be NULL")
	assert.Equal(t, 40.5884, p.Latitude.Float64)
	assert.Equal(t, -111.6386, p.Longitude.Float64)
}
