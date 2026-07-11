package llm_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/abteilung6/assetagent/internal/llm"
)

func TestRegistry_DefaultOllama(t *testing.T) {
	registry, err := llm.NewRegistry(llm.RegistryConfig{
		DefaultProvider: llm.ProviderOllama,
		Providers:       []llm.ProviderID{llm.ProviderOllama},
		OllamaBaseURL:   "http://localhost:11434",
		OllamaModel:     "llama3.2",
	})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	provider, err := registry.Default(context.Background())
	if err != nil {
		t.Fatalf("Default() error = %v", err)
	}
	if provider.Model() != "llama3.2" {
		t.Fatalf("Model() = %q, want llama3.2", provider.Model())
	}
}

func TestRegistry_DefaultNotInProviders(t *testing.T) {
	_, err := llm.NewRegistry(llm.RegistryConfig{
		DefaultProvider: llm.ProviderOpenRouter,
		Providers:       []llm.ProviderID{llm.ProviderOllama},
	})
	if err == nil {
		t.Fatal("NewRegistry() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "LLM_DEFAULT_PROVIDER") {
		t.Fatalf("error = %v, want default provider validation", err)
	}
}

func TestRegistry_DisabledProvider(t *testing.T) {
	registry, err := llm.NewRegistry(llm.RegistryConfig{
		DefaultProvider: llm.ProviderOllama,
		Providers:       []llm.ProviderID{llm.ProviderOllama},
	})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	_, err = registry.Resolve(context.Background(), llm.ProviderOpenRouter, "")
	if err == nil {
		t.Fatal("Resolve() error = nil, want disabled provider error")
	}
	if !errors.Is(err, llm.ErrProviderDisabled) {
		t.Fatalf("Resolve() error = %v, want ErrProviderDisabled", err)
	}
}

func TestRegistry_OpenRouterRequiresKey(t *testing.T) {
	registry, err := llm.NewRegistry(llm.RegistryConfig{
		DefaultProvider:        llm.ProviderOpenRouter,
		Providers:              []llm.ProviderID{llm.ProviderOpenRouter},
		OpenRouterDefaultModel: "openai/gpt-4o-mini",
	})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	_, err = registry.Resolve(context.Background(), llm.ProviderOpenRouter, "")
	if err == nil {
		t.Fatal("Resolve() error = nil, want missing api key error")
	}
	if !errors.Is(err, llm.ErrOpenRouterNoKey) {
		t.Fatalf("Resolve() error = %v, want ErrOpenRouterNoKey", err)
	}
}

func TestRegistry_OpenRouterModelAllowlist(t *testing.T) {
	registry, err := llm.NewRegistry(llm.RegistryConfig{
		DefaultProvider:          llm.ProviderOpenRouter,
		Providers:                []llm.ProviderID{llm.ProviderOpenRouter},
		OpenRouterAPIKey:         "test-key",
		OpenRouterDefaultModel:   "openai/gpt-4o-mini",
		OpenRouterModelAllowlist: []string{"openai/gpt-4o-mini"},
	})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	_, err = registry.Resolve(context.Background(), llm.ProviderOpenRouter, "anthropic/claude-3.5-sonnet")
	if err == nil {
		t.Fatal("Resolve() error = nil, want model not allowed error")
	}
	if !errors.Is(err, llm.ErrModelNotAllowed) {
		t.Fatalf("Resolve() error = %v, want ErrModelNotAllowed", err)
	}
}

func TestParseProviderID(t *testing.T) {
	id, err := llm.ParseProviderID(" OpenRouter ")
	if err != nil {
		t.Fatalf("ParseProviderID() error = %v", err)
	}
	if id != llm.ProviderOpenRouter {
		t.Fatalf("provider = %q, want openrouter", id)
	}
}

func TestParseProviderList_ignoresInvalid(t *testing.T) {
	list := llm.ParseProviderList("ollama, bogus, openrouter")
	if len(list) != 2 {
		t.Fatalf("len(list) = %d, want 2", len(list))
	}
}
