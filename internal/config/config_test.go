package config

import (
	"log/slog"
	"strings"
	"testing"
)

func TestLoad_defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("LOG_FORMAT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DatabaseURL != "" {
		t.Errorf("DatabaseURL = %q, want empty", cfg.DatabaseURL)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want info", cfg.LogLevel)
	}
	if cfg.LogFormat != "text" {
		t.Errorf("LogFormat = %q, want text", cfg.LogFormat)
	}
}

func TestLoad_overrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/assetagent?sslmode=disable")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "json")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DatabaseURL != "postgres://user:pass@localhost:5432/assetagent?sslmode=disable" {
		t.Errorf("DatabaseURL = %q, want postgres URL", cfg.DatabaseURL)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Errorf("LogLevel = %v, want debug", cfg.LogLevel)
	}
	if cfg.LogFormat != "json" {
		t.Errorf("LogFormat = %q, want json", cfg.LogFormat)
	}
}

func TestLoad_validationErrors(t *testing.T) {
	tests := []struct {
		name   string
		env    map[string]string
		errMsg string
	}{
		{
			name:   "invalid log level",
			env:    map[string]string{"LOG_LEVEL": "trace"},
			errMsg: "LOG_LEVEL",
		},
		{
			name:   "invalid log format",
			env:    map[string]string{"LOG_FORMAT": "yaml"},
			errMsg: "LOG_FORMAT",
		},
		{
			name:   "invalid database scheme",
			env:    map[string]string{"DATABASE_URL": "mysql://localhost/db"},
			errMsg: "scheme must be postgres",
		},
		{
			name:   "database url missing host",
			env:    map[string]string{"DATABASE_URL": "postgres:///dbname"},
			errMsg: "host is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "")
			t.Setenv("LOG_LEVEL", "")
			t.Setenv("LOG_FORMAT", "")

			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			_, err := Load()
			if err == nil {
				t.Fatal("Load() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Load() error = %q, want substring %q", err.Error(), tt.errMsg)
			}
		})
	}
}
