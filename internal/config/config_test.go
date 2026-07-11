package config

import (
	"log/slog"
	"strings"
	"testing"
)

func TestLoad_defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("API_ADDR", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("LOG_FORMAT", "")
	t.Setenv("OLLAMA_BASE_URL", "")
	t.Setenv("OLLAMA_MODEL", "")
	t.Setenv("LLM_DEFAULT_PROVIDER", "")
	t.Setenv("LLM_PROVIDERS", "")
	t.Setenv("OPENROUTER_BASE_URL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DatabaseURL != "" {
		t.Errorf("DatabaseURL = %q, want empty", cfg.DatabaseURL)
	}
	if cfg.APIAddr != ":8080" {
		t.Errorf("APIAddr = %q, want :8080", cfg.APIAddr)
	}
	if cfg.OllamaBaseURL != "http://localhost:11434" {
		t.Errorf("OllamaBaseURL = %q, want http://localhost:11434", cfg.OllamaBaseURL)
	}
	if cfg.OllamaModel != "llama3.2" {
		t.Errorf("OllamaModel = %q, want llama3.2", cfg.OllamaModel)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Errorf("LogLevel = %v, want info", cfg.LogLevel)
	}
	if cfg.LogFormat != "text" {
		t.Errorf("LogFormat = %q, want text", cfg.LogFormat)
	}
	if cfg.LLMDefaultProvider != "ollama" {
		t.Errorf("LLMDefaultProvider = %q, want ollama", cfg.LLMDefaultProvider)
	}
	if cfg.LLMProviders != "ollama" {
		t.Errorf("LLMProviders = %q, want ollama", cfg.LLMProviders)
	}
	if cfg.OpenRouterBaseURL != "https://openrouter.ai/api/v1" {
		t.Errorf("OpenRouterBaseURL = %q, want https://openrouter.ai/api/v1", cfg.OpenRouterBaseURL)
	}
}

func TestLoad_overrides(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/assetagent?sslmode=disable")
	t.Setenv("API_ADDR", ":9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "json")
	t.Setenv("OLLAMA_BASE_URL", "http://127.0.0.1:11434")
	t.Setenv("OLLAMA_MODEL", "qwen2.5:7b")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DatabaseURL != "postgres://user:pass@localhost:5432/assetagent?sslmode=disable" {
		t.Errorf("DatabaseURL = %q, want postgres URL", cfg.DatabaseURL)
	}
	if cfg.APIAddr != ":9090" {
		t.Errorf("APIAddr = %q, want :9090", cfg.APIAddr)
	}
	if cfg.OllamaBaseURL != "http://127.0.0.1:11434" {
		t.Errorf("OllamaBaseURL = %q, want http://127.0.0.1:11434", cfg.OllamaBaseURL)
	}
	if cfg.OllamaModel != "qwen2.5:7b" {
		t.Errorf("OllamaModel = %q, want qwen2.5:7b", cfg.OllamaModel)
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
		{
			name:   "invalid ollama base url scheme",
			env:    map[string]string{"OLLAMA_BASE_URL": "ftp://localhost:11434"},
			errMsg: "OLLAMA_BASE_URL",
		},
		{
			name:   "invalid llm default provider",
			env:    map[string]string{"LLM_DEFAULT_PROVIDER": "anthropic"},
			errMsg: "LLM provider",
		},
		{
			name:   "invalid llm providers list",
			env:    map[string]string{"LLM_PROVIDERS": "ollama,unknown"},
			errMsg: "LLM provider",
		},
		{
			name:   "invalid openrouter base url scheme",
			env:    map[string]string{"OPENROUTER_BASE_URL": "ftp://openrouter.ai"},
			errMsg: "OPENROUTER_BASE_URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "")
			t.Setenv("API_ADDR", "")
			t.Setenv("LOG_LEVEL", "")
			t.Setenv("LOG_FORMAT", "")
			t.Setenv("OLLAMA_BASE_URL", "")
			t.Setenv("OLLAMA_MODEL", "")
			t.Setenv("LLM_DEFAULT_PROVIDER", "")
			t.Setenv("LLM_PROVIDERS", "")
			t.Setenv("OPENROUTER_BASE_URL", "")

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
