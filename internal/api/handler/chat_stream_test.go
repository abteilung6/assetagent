package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/chat"
	"github.com/go-chi/chi/v5"
)

func TestPostChatStream_returnsSSE(t *testing.T) {
	chatSvc := &stubChatService{
		result: chat.Result{Answer: "You spent 15.98 EUR in December."},
	}

	router := chi.NewRouter()
	gen.HandlerWithOptions(handler.New(&stubListService{}, chatSvc, nil, nil, nil, nil, nil), gen.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handler.APIErrorHandler,
	})

	body := `{"messages":[{"role":"user","content":"How much did I spend in December?"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat/stream", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("content-type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "event: done") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestPostChatStream_rejectsEmptyMessages(t *testing.T) {
	chatSvc := &stubChatService{}

	router := chi.NewRouter()
	gen.HandlerWithOptions(handler.New(&stubListService{}, chatSvc, nil, nil, nil, nil, nil), gen.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handler.APIErrorHandler,
	})

	body := `{"messages":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/chat/stream", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
