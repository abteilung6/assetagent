package llm_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/abteilung6/assetagent/internal/llm"
)

func TestOllama_StreamComplete_text(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"Hello"},"done":false}` + "\n"))
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":" world"},"done":false}` + "\n"))
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":""},"done":true}` + "\n"))
	}))
	t.Cleanup(srv.Close)

	client := llm.NewOllama(srv.URL, "llama3.2")
	var deltas []string
	resp, err := client.StreamComplete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "Hi"}},
	}, func(content string) error {
		deltas = append(deltas, content)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamComplete() error = %v", err)
	}
	if strings.Join(deltas, "") != "Hello world" {
		t.Fatalf("deltas = %v", deltas)
	}
	if resp.Content != "" {
		t.Fatalf("content = %q, want empty final content", resp.Content)
	}
}

func TestOpenRouter_StreamComplete_text(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	t.Cleanup(srv.Close)

	client := llm.NewOpenRouter(srv.URL, "test-key", "openai/gpt-4o-mini", "", "")
	var deltas []string
	resp, err := client.StreamComplete(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: "Hi"}},
	}, func(content string) error {
		deltas = append(deltas, content)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamComplete() error = %v", err)
	}
	if strings.Join(deltas, "") != "Hello world" {
		t.Fatalf("deltas = %v", deltas)
	}
	if resp.Content != "Hello world" {
		t.Fatalf("content = %q", resp.Content)
	}
}
