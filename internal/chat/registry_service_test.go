package chat_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/abteilung6/assetagent/internal/chat"
	"github.com/abteilung6/assetagent/internal/llm"
)

func TestRegistryService_Chat_resolvesProvider(t *testing.T) {
	registry, err := llm.NewRegistry(llm.RegistryConfig{
		DefaultProvider: llm.ProviderOllama,
		Providers:       []llm.ProviderID{llm.ProviderOllama},
		OllamaBaseURL:   "http://127.0.0.1:1",
		OllamaModel:     "gemma4:12b",
	})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	tools := &fakeTools{
		result: json.RawMessage(`{"ok":true}`),
	}
	svc := chat.NewRegistryService(registry, tools, chat.DefaultConfig())

	// Will fail at Complete because Ollama isn't running, but proves resolve path.
	_, err = svc.Chat(context.Background(), "ollama", "gemma4:12b", []chat.Message{{
		Role:    llm.RoleUser,
		Content: "Hi",
	}})
	if err == nil {
		t.Fatal("Chat() error = nil, want completion failure")
	}
}

func TestRegistryService_Chat_rejectsUnknownProvider(t *testing.T) {
	registry, err := llm.NewRegistry(llm.RegistryConfig{
		DefaultProvider: llm.ProviderOllama,
		Providers:       []llm.ProviderID{llm.ProviderOllama},
	})
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	svc := chat.NewRegistryService(registry, &fakeTools{}, chat.DefaultConfig())
	_, err = svc.Chat(context.Background(), "anthropic", "", []chat.Message{{
		Role:    llm.RoleUser,
		Content: "Hi",
	}})
	if err == nil {
		t.Fatal("Chat() error = nil, want unknown provider")
	}
	if !errors.Is(err, llm.ErrProviderUnknown) {
		t.Fatalf("error = %v", err)
	}
}
