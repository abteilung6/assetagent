package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/abteilung6/assetagent/internal/api/middleware"
	"github.com/abteilung6/assetagent/internal/authctx"
	"github.com/google/uuid"
)

func TestRequireAuthMiddleware_publicPaths(t *testing.T) {
	h := middleware.RequireAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, path := range []string{"/api/health", "/health", "/auth/google/start", "/auth/logout"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", path, rec.Code)
		}
	}
}

func TestRequireAuthMiddleware_apiRequiresUser(t *testing.T) {
	h := middleware.RequireAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/transactions", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/transactions", nil)
	req2 = req2.WithContext(authctx.WithUserID(req2.Context(), uuid.New()))
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("authenticated status = %d, want 200", rec2.Code)
	}
}

func TestCORSMiddleware_exactOrigin(t *testing.T) {
	h := middleware.CORSMiddleware("http://localhost:5173, https://app.example.com")(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("Allow-Origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Allow-Credentials = %q", got)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req2.Header.Set("Origin", "http://evil.example")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if got := rec2.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected Allow-Origin for evil origin: %q", got)
	}

	opt := httptest.NewRequest(http.MethodOptions, "/api/me", nil)
	opt.Header.Set("Origin", "http://localhost:5173")
	optRec := httptest.NewRecorder()
	h.ServeHTTP(optRec, opt)
	if optRec.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d, want 204", optRec.Code)
	}
}
