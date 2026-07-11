package llm_test

import (
	"testing"

	"github.com/abteilung6/assetagent/internal/llm"
)

func TestRegistry_ModelCatalog_openRouterOnly(t *testing.T) {
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

	catalog := registry.ModelCatalog()
	if len(catalog.Options) != 1 {
		t.Fatalf("len(options) = %d, want 1", len(catalog.Options))
	}
	if catalog.Options[0].Group != "" {
		t.Fatalf("group = %q, want empty for single provider", catalog.Options[0].Group)
	}
	if catalog.Options[0].Label != "GPT-4o mini" {
		t.Fatalf("label = %q", catalog.Options[0].Label)
	}
}

func TestRegistry_ModelCatalog_devGroups(t *testing.T) {
	registry, err := llm.NewRegistry(llm.RegistryConfig{
		DefaultProvider:        llm.ProviderOllama,
		Providers:              []llm.ProviderID{llm.ProviderOllama, llm.ProviderOpenRouter},
		OllamaModel:            "gemma4:12b",
		OpenRouterAPIKey:       "test-key",
		OpenRouterDefaultModel: "openai/gpt-4o-mini",
	})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	catalog := registry.ModelCatalog()
	if len(catalog.Options) != 2 {
		t.Fatalf("len(options) = %d, want 2", len(catalog.Options))
	}
	if catalog.Options[0].Group != llm.GroupLocal {
		t.Fatalf("ollama group = %q", catalog.Options[0].Group)
	}
	if catalog.Options[1].Group != llm.GroupCloud {
		t.Fatalf("openrouter group = %q", catalog.Options[1].Group)
	}
}

func TestRegistry_ModelCatalog_skipsUnconfiguredOpenRouter(t *testing.T) {
	registry, err := llm.NewRegistry(llm.RegistryConfig{
		DefaultProvider: llm.ProviderOllama,
		Providers:       []llm.ProviderID{llm.ProviderOllama, llm.ProviderOpenRouter},
		OllamaModel:     "gemma4:12b",
	})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	catalog := registry.ModelCatalog()
	if len(catalog.Options) != 1 {
		t.Fatalf("len(options) = %d, want 1", len(catalog.Options))
	}
	if catalog.Options[0].Provider != llm.ProviderOllama {
		t.Fatalf("provider = %q", catalog.Options[0].Provider)
	}
}

func TestModelLabel_knownAndFallback(t *testing.T) {
	if got := llm.ModelLabel("openai/gpt-4o-mini"); got != "GPT-4o mini" {
		t.Fatalf("label = %q", got)
	}
	if got := llm.ModelLabel("vendor/some-model_v2"); got == "" {
		t.Fatal("expected fallback label")
	}
}
