package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/go-chi/chi/v5"
)

type noopList struct{}

func (noopList) ListTransactions(context.Context, domain.ListParams) (domain.ListResult, error) {
	return domain.ListResult{}, nil
}

func TestGetHealth(t *testing.T) {
	router := chi.NewRouter()
	gen.HandlerFromMux(handler.New(noopList{}, nil, nil, nil, nil, nil, nil, nil), router)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	var resp gen.HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Status != "ok" {
		t.Fatalf("status = %q, want ok", resp.Status)
	}
}
