package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/chat"
	"github.com/go-chi/chi/v5"
)

func TestPostChat_rejectsInvalidContext(t *testing.T) {
	chatSvc := &stubChatService{
		result: chat.Result{Answer: "ok"},
	}

	router := chi.NewRouter()
	gen.HandlerWithOptions(handler.New(&stubListService{}, chatSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil), gen.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handler.APIErrorHandler,
	})

	body := `{"messages":[{"role":"user","content":"hi"}],"context":{"baseline_id":"not-a-uuid"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestPostChat_acceptsValidContext(t *testing.T) {
	chatSvc := &stubChatService{
		result: chat.Result{Answer: "ok", ToolCalls: nil},
	}

	router := chi.NewRouter()
	gen.HandlerWithOptions(handler.New(&stubListService{}, chatSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil), gen.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handler.APIErrorHandler,
	})

	body := `{"messages":[{"role":"user","content":"hi"}],"context":{"route":"/baseline","yyyy_mm":"2026-03","from":"2026-03-01","to":"2026-03-31"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp gen.ChatResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Answer != "ok" {
		t.Fatalf("answer = %q", resp.Answer)
	}
}
