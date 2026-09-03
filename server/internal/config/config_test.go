package config_test

import (
	"testing"

	"github.com/MattSilvaa/powhunter/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Production must not silently fall back to a well-known database credential,
// which is what the previous implementation did.
func TestProductionRequiresExplicitCredentials(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("ALLOWED_ORIGINS", "https://powhunter.app")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")

	_, err := config.Load()

	require.ErrorIs(t, err, config.ErrMissingConfig)
	assert.Contains(t, err.Error(), "DB_USER")
	assert.Contains(t, err.Error(), "DB_PASSWORD")
}

func TestProductionRejectsWildcardOrigin(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("DB_USER", "app")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("ALLOWED_ORIGINS", "*")

	_, err := config.Load()

	require.ErrorIs(t, err, config.ErrMissingConfig)
	assert.Contains(t, err.Error(), "ALLOWED_ORIGINS")
}

func TestProductionDefaultsToTLSForDatabase(t *testing.T) {
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("DB_USER", "app")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("ALLOWED_ORIGINS", "https://powhunter.app")

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, "require", cfg.Database.SSLMode)
	assert.True(t, cfg.IsProduction())
}

func TestDevelopmentKeepsConvenientDefaults(t *testing.T) {
	t.Setenv("ENVIRONMENT", "development")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("ALLOWED_ORIGINS", "")

	cfg, err := config.Load()

	require.NoError(t, err)
	assert.Equal(t, "powhunter_rw", cfg.Database.User)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
	assert.False(t, cfg.IsProduction())
}
