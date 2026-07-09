package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/abteilung6/assetagent/internal/llm"
)

func TestOllama_Ping_ok(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Fatalf("path = %s, want /api/tags", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"models":[{"name":"llama3.2:latest"}]}`))
	}))
	t.Cleanup(srv.Close)

	client := llm.NewOllama(srv.URL, "llama3.2")
	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func TestOllama_Ping_missingModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"models":[{"name":"other:latest"}]}`))
	}))
	t.Cleanup(srv.Close)

	client := llm.NewOllama(srv.URL, "llama3.2")
	err := client.Ping(context.Background())
	if err == nil {
		t.Fatal("Ping() error = nil, want missing model error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Ping() error = %v, want not found", err)
	}
}

func TestOllama_Complete_text(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Fatalf("path = %s, want /api/chat", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}

		var req struct {
			Model  string `json:"model"`
			Stream bool   `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "llama3.2" || req.Stream {
			t.Fatalf("request = %+v, want model llama3.2 stream false", req)
		}

		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"42.00 EUR"}}`))
	}))
	t.Cleanup(srv.Close)

	client := llm.NewOllama(srv.URL, "llama3.2")
	resp, err := client.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "How much?"}},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if resp.Content != "42.00 EUR" {
		t.Fatalf("Content = %q, want 42.00 EUR", resp.Content)
	}
	if len(resp.ToolCalls) != 0 {
		t.Fatalf("ToolCalls = %+v, want empty", resp.ToolCalls)
	}
}

func TestOllama_Complete_toolCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"message":{
				"role":"assistant",
				"content":"",
				"tool_calls":[
					{"function":{"name":"get_monthly_cashflow","arguments":{"from":"2025-12-01","to":"2025-12-31"}}}
				]
			}
		}`))
	}))
	t.Cleanup(srv.Close)

	client := llm.NewOllama(srv.URL, "llama3.2")
	resp, err := client.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "December spending?"}},
		Tools: []llm.Tool{{
			Name:        "get_monthly_cashflow",
			Description: "Monthly cashflow",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"from":{"type":"string"},"to":{"type":"string"}}}`),
		}},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "get_monthly_cashflow" {
		t.Fatalf("tool name = %q", resp.ToolCalls[0].Name)
	}
}
