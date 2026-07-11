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

func TestOpenRouter_Ping_ok(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Fatalf("path = %s, want /models", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Fatalf("Authorization = %q, want Bearer test-key", auth)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"openai/gpt-4o-mini"}]}`))
	}))
	t.Cleanup(srv.Close)

	client := llm.NewOpenRouter(srv.URL, "test-key", "openai/gpt-4o-mini", "https://example.com", "assetagent")
	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}

func TestOpenRouter_Ping_missingModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"id":"other/model"}]}`))
	}))
	t.Cleanup(srv.Close)

	client := llm.NewOpenRouter(srv.URL, "test-key", "openai/gpt-4o-mini", "", "")
	err := client.Ping(context.Background())
	if err == nil {
		t.Fatal("Ping() error = nil, want missing model error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Ping() error = %v, want not found", err)
	}
}

func TestOpenRouter_Complete_text(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %s, want /chat/completions", r.URL.Path)
		}

		var req struct {
			Model           string `json:"model"`
			Stream          bool   `json:"stream"`
			ReasoningEffort string `json:"reasoning_effort"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "openai/gpt-4o-mini" || req.Stream {
			t.Fatalf("request = %+v, want model openai/gpt-4o-mini stream false", req)
		}
		if req.ReasoningEffort != "none" {
			t.Fatalf("reasoning_effort = %q, want none", req.ReasoningEffort)
		}

		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"42.00 EUR"}}]}`))
	}))
	t.Cleanup(srv.Close)

	client := llm.NewOpenRouter(srv.URL, "test-key", "openai/gpt-4o-mini", "", "")
	resp, err := client.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "How much?"}},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if resp.Content != "42.00 EUR" {
		t.Fatalf("Content = %q, want 42.00 EUR", resp.Content)
	}
}

func TestOpenRouter_Complete_toolLoopArgumentsAreStrings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Role      string `json:"role"`
				ToolCalls []struct {
					Function struct {
						Name      string          `json:"name"`
						Arguments json.RawMessage `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		for _, msg := range req.Messages {
			for _, call := range msg.ToolCalls {
				if len(call.Function.Arguments) == 0 {
					continue
				}
				if call.Function.Arguments[0] != '"' {
					t.Fatalf("arguments = %s, want JSON string", call.Function.Arguments)
				}
			}
		}

		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"15.98 EUR"}}]}`))
	}))
	t.Cleanup(srv.Close)

	client := llm.NewOpenRouter(srv.URL, "test-key", "openai/gpt-4o-mini", "", "")
	_, err := client.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: "December spending?"},
			{
				Role: llm.RoleAssistant,
				ToolCalls: []llm.ToolCall{{
					ID:        "call_1",
					Name:      "get_cashflow",
					Arguments: json.RawMessage(`{"from":"2025-12-01","to":"2025-12-31"}`),
				}},
			},
			{
				Role:       llm.RoleTool,
				ToolCallID: "call_1",
				Content:    `{"ok":true,"expenses":"15.98","currency":"EUR"}`,
			},
		},
	})
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
}

func TestOpenRouter_Complete_toolCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"choices":[{
				"message":{
					"role":"assistant",
					"content":"",
					"tool_calls":[{
						"id":"call_1",
						"type":"function",
						"function":{"name":"get_cashflow","arguments":"{\"from\":\"2025-12-01\",\"to\":\"2025-12-31\"}"}
					}]
				}
			}]
		}`))
	}))
	t.Cleanup(srv.Close)

	client := llm.NewOpenRouter(srv.URL, "test-key", "openai/gpt-4o-mini", "", "")
	resp, err := client.Complete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "December spending?"}},
		Tools: []llm.Tool{{
			Name:        "get_cashflow",
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
	if resp.ToolCalls[0].ID != "call_1" {
		t.Fatalf("tool id = %q, want call_1", resp.ToolCalls[0].ID)
	}
	if resp.ToolCalls[0].Name != "get_cashflow" {
		t.Fatalf("tool name = %q", resp.ToolCalls[0].Name)
	}
}
