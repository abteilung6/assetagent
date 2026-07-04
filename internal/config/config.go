package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL string
	LogLevel    slog.Level
	LogFormat   string
}

func Load() (Config, error) {
	level, err := parseLogLevel(envOrDefault("LOG_LEVEL", "info"))
	if err != nil {
		return Config{}, err
	}

	format := envOrDefault("LOG_FORMAT", "text")
	if format != "text" && format != "json" {
		return Config{}, fmt.Errorf("LOG_FORMAT must be text or json, got %q", format)
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL != "" {
		if err := validateDatabaseURL(databaseURL); err != nil {
			return Config{}, err
		}
	}

	return Config{
		DatabaseURL: databaseURL,
		LogLevel:    level,
		LogFormat:   format,
	}, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseLogLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(raw) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error, got %q", raw)
	}
}

func validateDatabaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("DATABASE_URL: %w", err)
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return fmt.Errorf("DATABASE_URL: scheme must be postgres or postgresql, got %q", u.Scheme)
	}
	if u.Host == "" {
		return errors.New("DATABASE_URL: host is required")
	}
	return nil
}
