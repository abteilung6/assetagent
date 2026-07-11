package main

import (
	"fmt"
	"strings"

	"github.com/abteilung6/assetagent/internal/config"
	"github.com/abteilung6/assetagent/internal/llm"
)

func newLLMRegistry(cfg config.Config) (*llm.Registry, error) {
	defaultProvider, err := llm.ParseProviderID(cfg.LLMDefaultProvider)
	if err != nil {
		return nil, err
	}

	providers := llm.ParseProviderList(cfg.LLMProviders)
	if len(providers) == 0 {
		return nil, fmt.Errorf("LLM_PROVIDERS must list at least one provider")
	}

	return llm.NewRegistry(llm.RegistryConfig{
		DefaultProvider:          defaultProvider,
		Providers:                providers,
		OllamaBaseURL:            cfg.OllamaBaseURL,
		OllamaModel:              cfg.OllamaModel,
		OpenRouterBaseURL:        cfg.OpenRouterBaseURL,
		OpenRouterAPIKey:         cfg.OpenRouterAPIKey,
		OpenRouterDefaultModel:   cfg.OpenRouterDefaultModel,
		OpenRouterModelAllowlist: splitCommaList(cfg.OpenRouterModelAllowlist),
		OpenRouterAppURL:         cfg.OpenRouterAppURL,
		OpenRouterAppName:        cfg.OpenRouterAppName,
	})
}

func splitCommaList(raw string) []string {
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
