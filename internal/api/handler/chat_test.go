package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/chat"
	"github.com/abteilung6/assetagent/internal/llm"
	"github.com/go-chi/chi/v5"
)

type stubChatService struct {
	result chat.Result
	err    error
	input  []chat.Message
}

func (s *stubChatService) Chat(ctx context.Context, provider, model string, messages []chat.Message) (chat.Result, error) {
	s.input = messages
	return s.result, s.err
}

func (s *stubChatService) StreamChat(
	ctx context.Context,
	provider, model string,
	messages []chat.Message,
	write chat.StreamWriter,
) error {
	s.input = messages
	if s.err != nil {
		return write(chat.StreamEventError, map[string]string{"message": s.err.Error()})
	}
	return write(chat.StreamEventDone, map[string]any{
		"answer":     s.result.Answer,
		"tool_calls": s.result.ToolCalls,
	})
}

func TestPostChat_returnsGroundedAnswer(t *testing.T) {
	chatSvc := &stubChatService{
		result: chat.Result{
			Answer: "You spent 15.98 EUR in December.",
			ToolCalls: []chat.ToolCall{{
				Name:   "get_cashflow",
				Input:  json.RawMessage(`{"from":"2025-12-01","to":"2025-12-31"}`),
				Result: json.RawMessage(`{"income":"0","expenses":"15.98","net":"-15.98","currency":"EUR"}`),
			}},
		},
	}

	router := chi.NewRouter()
	gen.HandlerWithOptions(handler.New(&stubListService{}, chatSvc, nil, nil, nil, nil, nil, nil), gen.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handler.APIErrorHandler,
	})

	body := `{"messages":[{"role":"user","content":"How much did I spend in December?"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp gen.ChatResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Answer != "You spent 15.98 EUR in December." {
		t.Fatalf("answer = %q", resp.Answer)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("tool calls = %+v", resp.ToolCalls)
	}
	if resp.ToolCalls[0].Name != "get_cashflow" {
		t.Fatalf("tool name = %q", resp.ToolCalls[0].Name)
	}
	if len(chatSvc.input) != 1 || chatSvc.input[0].Role != llm.RoleUser {
		t.Fatalf("input = %+v", chatSvc.input)
	}
}

func TestPostChat_rejectsEmptyMessages(t *testing.T) {
	chatSvc := &stubChatService{}

	router := chi.NewRouter()
	gen.HandlerWithOptions(handler.New(&stubListService{}, chatSvc, nil, nil, nil, nil, nil, nil), gen.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handler.APIErrorHandler,
	})

	body := `{"messages":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
