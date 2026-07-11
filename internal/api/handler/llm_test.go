package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/llm"
	"github.com/go-chi/chi/v5"
)

func TestGetLLMModels_returnsCatalog(t *testing.T) {
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

	router := chi.NewRouter()
	gen.HandlerWithOptions(handler.New(&stubListService{}, nil, registry), gen.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handler.APIErrorHandler,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/llm/models", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp gen.LLMModelCatalog
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Options) != 2 {
		t.Fatalf("options = %+v", resp.Options)
	}
	if resp.Default.Provider != gen.Ollama {
		t.Fatalf("default provider = %q", resp.Default.Provider)
	}
	if resp.Options[0].Label == "" {
		t.Fatal("expected human label")
	}
}
