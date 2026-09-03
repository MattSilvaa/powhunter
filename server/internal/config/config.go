// Package config loads and validates runtime configuration from the environment.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPort            = "8080"
	defaultDBPort          = "5432"
	defaultDBName          = "powhunter"
	defaultDBHost          = "localhost"
	defaultSSLMode         = "disable"
	productionSSLMode      = "require"
	defaultReadTimeout     = 15 * time.Second
	defaultWriteTimeout    = 15 * time.Second
	defaultIdleTimeout     = 60 * time.Second
	defaultHeaderTimeout   = 5 * time.Second
	defaultMaxBodyBytes    = 64 * 1024
	defaultRatePerSecond   = 10.0
	defaultRateBurst       = 20
	defaultWriteRatePerSec = 1.0
	defaultWriteRateBurst  = 5
	defaultAppBaseURL      = "http://localhost:3000"
	defaultMailFrom        = "Powhunter <noreply@powhunter.app>"
	defaultSupportEmail    = "support@powhunter.app"
	// Requesting a login link sends mail to an address the caller names, so it
	// gets a bucket far tighter than ordinary writes.
	defaultLoginRatePerSec = 0.2
	defaultLoginRateBurst  = 3
)

// ErrMissingConfig reports configuration that must be set in production.
var ErrMissingConfig = errors.New("missing required configuration")

// Database holds connection settings.
type Database struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// DSN renders the lib/pq connection string.
func (d Database) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

// Config is the resolved runtime configuration.
type Config struct {
	Environment       string
	Port              string
	AllowedOrigins    string
	TrustProxyHeaders bool
	MaxBodyBytes      int64
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	RatePerSecond     float64
	RateBurst         int
	WriteRatePerSec   float64
	WriteRateBurst    int
	LoginRatePerSec   float64
	LoginRateBurst    int
	AppBaseURL        string
	MailFrom          string
	SupportEmail      string
	ResendAPIKey      string
	Database          Database
}

// IsProduction reports whether this is a production deployment.
func (c Config) IsProduction() bool {
	return strings.EqualFold(c.Environment, "production")
}

// Load reads configuration from the environment.
//
// Defaults are development conveniences. In production the database
// credentials must be supplied explicitly: silently falling back to a
// well-known username and password, as this previously did, turns a
// misconfigured deploy into a security problem rather than a startup failure.
func Load() (Config, error) {
	environment := envOrDefault("ENVIRONMENT", "development")
	isProduction := strings.EqualFold(environment, "production")

	sslDefault := defaultSSLMode
	if isProduction {
		sslDefault = productionSSLMode
	}

	cfg := Config{
		Environment:       environment,
		Port:              envOrDefault("PORT", defaultPort),
		AllowedOrigins:    envOrDefault("ALLOWED_ORIGINS", "*"),
		TrustProxyHeaders: envBool("TRUST_PROXY_HEADERS", isProduction),
		MaxBodyBytes:      int64(envInt("MAX_BODY_BYTES", defaultMaxBodyBytes)),
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		ReadHeaderTimeout: defaultHeaderTimeout,
		RatePerSecond:     envFloat("RATE_LIMIT_PER_SECOND", defaultRatePerSecond),
		RateBurst:         envInt("RATE_LIMIT_BURST", defaultRateBurst),
		WriteRatePerSec:   envFloat("WRITE_RATE_LIMIT_PER_SECOND", defaultWriteRatePerSec),
		WriteRateBurst:    envInt("WRITE_RATE_LIMIT_BURST", defaultWriteRateBurst),
		LoginRatePerSec:   envFloat("LOGIN_RATE_LIMIT_PER_SECOND", defaultLoginRatePerSec),
		LoginRateBurst:    envInt("LOGIN_RATE_LIMIT_BURST", defaultLoginRateBurst),
		AppBaseURL:        strings.TrimRight(envOrDefault("APP_BASE_URL", defaultAppBaseURL), "/"),
		MailFrom:          envOrDefault("MAIL_FROM", defaultMailFrom),
		SupportEmail:      envOrDefault("SUPPORT_EMAIL", defaultSupportEmail),
		ResendAPIKey:      os.Getenv("RESEND_API_KEY"),
		Database: Database{
			Host:     envOrDefault("DB_HOST", defaultDBHost),
			Port:     envOrDefault("DB_PORT", defaultDBPort),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     envOrDefault("DB_NAME", defaultDBName),
			SSLMode:  envOrDefault("DB_SSLMODE", sslDefault),
		},
	}

	if isProduction {
		if err := cfg.validateProduction(); err != nil {
			return Config{}, err
		}
	} else {
		if cfg.Database.User == "" {
			cfg.Database.User = "powhunter_rw"
		}

		if cfg.Database.Password == "" {
			cfg.Database.Password = "powhunter_rw"
		}
	}

	return cfg, nil
}

func (c Config) validateProduction() error {
	var missing []string

	if c.Database.User == "" {
		missing = append(missing, "DB_USER")
	}

	if c.Database.Password == "" {
		missing = append(missing, "DB_PASSWORD")
	}

	if c.AllowedOrigins == "" || c.AllowedOrigins == "*" {
		missing = append(missing, "ALLOWED_ORIGINS (a wildcard is not allowed in production)")
	}

	// A production deploy left on the default would email login links pointing
	// at localhost, which no recipient can open.
	if c.AppBaseURL == "" || c.AppBaseURL == defaultAppBaseURL {
		missing = append(missing, "APP_BASE_URL (the default localhost value is not usable in production)")
	}

	// Without a mail provider the login link is only written to the server log,
	// so nobody could sign in.
	if c.ResendAPIKey == "" {
		missing = append(missing, "RESEND_API_KEY (required to deliver login links)")
	}

	if len(missing) > 0 {
		return fmt.Errorf("%w: %s", ErrMissingConfig, strings.Join(missing, ", "))
	}

	return nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func envFloat(key string, fallback float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}

	return parsed
}
