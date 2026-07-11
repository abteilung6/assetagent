package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type ProviderID string

const (
	ProviderOllama     ProviderID = "ollama"
	ProviderOpenRouter ProviderID = "openrouter"
)

var (
	ErrProviderDisabled  = errors.New("llm provider is disabled")
	ErrProviderUnknown   = errors.New("unknown llm provider")
	ErrModelNotAllowed   = errors.New("llm model is not allowed")
	ErrOpenRouterNoKey   = errors.New("openrouter api key is not configured")
)

type RegistryConfig struct {
	DefaultProvider ProviderID
	Providers       []ProviderID

	OllamaBaseURL string
	OllamaModel   string

	OpenRouterBaseURL      string
	OpenRouterAPIKey       string
	OpenRouterDefaultModel string
	OpenRouterModelAllowlist []string
	OpenRouterAppURL       string
	OpenRouterAppName      string
}

type Registry struct {
	cfg RegistryConfig
}

func NewRegistry(cfg RegistryConfig) (*Registry, error) {
	if cfg.DefaultProvider == "" {
		cfg.DefaultProvider = ProviderOllama
	}
	if len(cfg.Providers) == 0 {
		cfg.Providers = []ProviderID{ProviderOllama}
	}
	if cfg.OllamaBaseURL == "" {
		cfg.OllamaBaseURL = "http://localhost:11434"
	}
	if cfg.OllamaModel == "" {
		cfg.OllamaModel = "llama3.2"
	}
	if cfg.OpenRouterBaseURL == "" {
		cfg.OpenRouterBaseURL = defaultOpenRouterBaseURL
	}

	if !containsProvider(cfg.Providers, cfg.DefaultProvider) {
		return nil, fmt.Errorf("LLM_DEFAULT_PROVIDER %q is not listed in LLM_PROVIDERS", cfg.DefaultProvider)
	}

	return &Registry{cfg: cfg}, nil
}

func (r *Registry) DefaultProviderID() ProviderID {
	return r.cfg.DefaultProvider
}

func (r *Registry) EnabledProviders() []ProviderID {
	out := make([]ProviderID, len(r.cfg.Providers))
	copy(out, r.cfg.Providers)
	return out
}

func (r *Registry) OpenRouterConfigured() bool {
	return r.isEnabled(ProviderOpenRouter) && r.cfg.OpenRouterAPIKey != ""
}

func (r *Registry) Default(ctx context.Context) (Provider, error) {
	return r.Resolve(ctx, r.cfg.DefaultProvider, "")
}

func (r *Registry) Resolve(ctx context.Context, provider ProviderID, model string) (Provider, error) {
	if provider == "" {
		provider = r.cfg.DefaultProvider
	}

	if !r.isEnabled(provider) {
		return nil, fmt.Errorf("%w: %s", ErrProviderDisabled, provider)
	}

	switch provider {
	case ProviderOllama:
		resolvedModel := model
		if resolvedModel == "" {
			resolvedModel = r.cfg.OllamaModel
		}
		return NewOllama(r.cfg.OllamaBaseURL, resolvedModel), nil
	case ProviderOpenRouter:
		if r.cfg.OpenRouterAPIKey == "" {
			return nil, ErrOpenRouterNoKey
		}
		resolvedModel := model
		if resolvedModel == "" {
			resolvedModel = r.cfg.OpenRouterDefaultModel
		}
		if resolvedModel == "" {
			return nil, fmt.Errorf("openrouter model is required")
		}
		if err := r.validateOpenRouterModel(resolvedModel); err != nil {
			return nil, err
		}
		return NewOpenRouter(
			r.cfg.OpenRouterBaseURL,
			r.cfg.OpenRouterAPIKey,
			resolvedModel,
			r.cfg.OpenRouterAppURL,
			r.cfg.OpenRouterAppName,
		), nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrProviderUnknown, provider)
	}
}

func (r *Registry) validateOpenRouterModel(model string) error {
	if len(r.cfg.OpenRouterModelAllowlist) == 0 {
		return nil
	}
	for _, allowed := range r.cfg.OpenRouterModelAllowlist {
		if model == allowed {
			return nil
		}
	}
	return fmt.Errorf("%w: %q", ErrModelNotAllowed, model)
}

func (r *Registry) isEnabled(provider ProviderID) bool {
	return containsProvider(r.cfg.Providers, provider)
}

func containsProvider(providers []ProviderID, want ProviderID) bool {
	for _, provider := range providers {
		if provider == want {
			return true
		}
	}
	return false
}

func ParseProviderID(raw string) (ProviderID, error) {
	switch ProviderID(strings.ToLower(strings.TrimSpace(raw))) {
	case ProviderOllama:
		return ProviderOllama, nil
	case ProviderOpenRouter:
		return ProviderOpenRouter, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrProviderUnknown, raw)
	}
}

func ParseProviderList(raw string) []ProviderID {
	parts := strings.Split(raw, ",")
	out := make([]ProviderID, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := ParseProviderID(part)
		if err == nil {
			out = append(out, id)
		}
	}
	return out
}
