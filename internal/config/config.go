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
	DatabaseURL   string
	APIAddr       string
	OllamaBaseURL string
	OllamaModel   string
	LogLevel      slog.Level
	LogFormat     string

	LLMDefaultProvider       string
	LLMProviders             string
	OpenRouterBaseURL        string
	OpenRouterAPIKey         string
	OpenRouterDefaultModel   string
	OpenRouterModelAllowlist string
	OpenRouterAppURL         string
	OpenRouterAppName        string

	LangfuseEnabled     bool
	LangfusePublicKey   string
	LangfuseSecretKey   string
	OTLPEndpoint        string
	LangfuseTraceDetail string
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

	ollamaBaseURL := envOrDefault("OLLAMA_BASE_URL", "http://localhost:11434")
	if err := validateHTTPBaseURL(ollamaBaseURL, "OLLAMA_BASE_URL"); err != nil {
		return Config{}, err
	}

	openRouterBaseURL := envOrDefault("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1")
	if err := validateHTTPBaseURL(openRouterBaseURL, "OPENROUTER_BASE_URL"); err != nil {
		return Config{}, err
	}

	llmDefaultProvider := envOrDefault("LLM_DEFAULT_PROVIDER", "ollama")
	if _, err := parseLLMProvider(llmDefaultProvider); err != nil {
		return Config{}, err
	}

	llmProviders := envOrDefault("LLM_PROVIDERS", "ollama")
	for _, part := range splitCSV(llmProviders) {
		if _, err := parseLLMProvider(part); err != nil {
			return Config{}, err
		}
	}

	return Config{
		DatabaseURL:              databaseURL,
		APIAddr:                  envOrDefault("API_ADDR", ":8080"),
		OllamaBaseURL:              ollamaBaseURL,
		OllamaModel:                envOrDefault("OLLAMA_MODEL", "llama3.2"),
		LogLevel:                   level,
		LogFormat:                  format,
		LLMDefaultProvider:         llmDefaultProvider,
		LLMProviders:               llmProviders,
		OpenRouterBaseURL:          openRouterBaseURL,
		OpenRouterAPIKey:           os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterDefaultModel:     envOrDefault("OPENROUTER_DEFAULT_MODEL", ""),
		OpenRouterModelAllowlist:   envOrDefault("OPENROUTER_MODEL_ALLOWLIST", ""),
		OpenRouterAppURL:           envOrDefault("OPENROUTER_APP_URL", ""),
		OpenRouterAppName:          envOrDefault("OPENROUTER_APP_NAME", "assetagent"),
		LangfuseEnabled:            parseBoolEnv("LANGFUSE_ENABLED", false),
		LangfusePublicKey:          os.Getenv("LANGFUSE_PUBLIC_KEY"),
		LangfuseSecretKey:          os.Getenv("LANGFUSE_SECRET_KEY"),
		OTLPEndpoint:               envOrDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:3000/api/public/otel"),
		LangfuseTraceDetail:        envOrDefault("LANGFUSE_TRACE_DETAIL", "metadata_only"),
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

func validateHTTPBaseURL(raw, envName string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%s: %w", envName, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%s: scheme must be http or https, got %q", envName, u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("%s: host is required", envName)
	}
	return nil
}

func parseLLMProvider(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "ollama", "openrouter":
		return strings.ToLower(strings.TrimSpace(raw)), nil
	default:
		return "", fmt.Errorf("LLM provider must be ollama or openrouter, got %q", raw)
	}
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseBoolEnv(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
