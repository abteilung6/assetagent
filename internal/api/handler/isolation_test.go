package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/api/middleware"
	"github.com/abteilung6/assetagent/internal/authctx"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/go-chi/chi/v5"
)

func TestHouseholdIsolation_userCannotListOtherHouseholdTransactions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	t.Cleanup(pool.Close)

	auth := repository.NewAuth(pool)
	sessions := service.NewSession(auth, service.SessionConfig{
		CookieName: "session",
		Idle:       2 * time.Hour,
		Absolute:   24 * time.Hour,
	})

	userA, err := auth.CreateUser(ctx, repository.CreateUserInput{DisplayName: "User A"})
	if err != nil {
		t.Fatalf("CreateUser A: %v", err)
	}
	houseA, err := auth.CreateHousehold(ctx, "House A")
	if err != nil {
		t.Fatalf("CreateHousehold A: %v", err)
	}
	if _, err := auth.CreateMembership(ctx, houseA.ID, userA.ID, domain.MembershipRoleOwner); err != nil {
		t.Fatalf("membership A: %v", err)
	}

	userB, err := auth.CreateUser(ctx, repository.CreateUserInput{DisplayName: "User B"})
	if err != nil {
		t.Fatalf("CreateUser B: %v", err)
	}
	houseB, err := auth.CreateHousehold(ctx, "House B")
	if err != nil {
		t.Fatalf("CreateHousehold B: %v", err)
	}
	if _, err := auth.CreateMembership(ctx, houseB.ID, userB.ID, domain.MembershipRoleOwner); err != nil {
		t.Fatalf("membership B: %v", err)
	}

	importer := service.NewImport(pool)
	samplePath := filepath.Join("..", "..", "..", "testdata", "sparkasse", "sample.csv")

	ctxA := authctx.WithHouseholdID(ctx, houseA.ID)
	result, err := importer.ImportFile(ctxA, samplePath, domain.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportFile A: %v", err)
	}
	if result.Inserted == 0 {
		t.Fatal("expected transactions imported for household A")
	}

	rawA, _, err := sessions.IssueSession(ctx, userA.ID, "test", nil)
	if err != nil {
		t.Fatalf("IssueSession A: %v", err)
	}
	rawB, _, err := sessions.IssueSession(ctx, userB.ID, "test", nil)
	if err != nil {
		t.Fatalf("IssueSession B: %v", err)
	}

	txRepo := repository.NewTransaction(pool)
	router := chi.NewRouter()
	router.Use(middleware.ResolveSessionMiddleware(sessions))
	router.Use(middleware.RequireAuthMiddleware)
	gen.HandlerWithOptions(
		handler.New(service.NewList(txRepo), nil, nil, importer, nil, nil, nil, nil, nil, nil, nil, nil, sessions),
		gen.ChiServerOptions{
			BaseRouter:       router,
			ErrorHandlerFunc: handler.APIErrorHandler,
		},
	)

	reqA := httptest.NewRequest(http.MethodGet, "/api/transactions?limit=50", nil)
	reqA.AddCookie(&http.Cookie{Name: "session", Value: rawA})
	recA := httptest.NewRecorder()
	router.ServeHTTP(recA, reqA)
	if recA.Code != http.StatusOK {
		t.Fatalf("user A status = %d body=%s", recA.Code, recA.Body.String())
	}
	var listA gen.TransactionListResponse
	if err := json.NewDecoder(recA.Body).Decode(&listA); err != nil {
		t.Fatalf("decode A: %v", err)
	}
	if listA.Pagination.Total != int64(result.Inserted) {
		t.Fatalf("user A total = %d, want %d", listA.Pagination.Total, result.Inserted)
	}

	reqB := httptest.NewRequest(http.MethodGet, "/api/transactions?limit=50", nil)
	reqB.AddCookie(&http.Cookie{Name: "session", Value: rawB})
	recB := httptest.NewRecorder()
	router.ServeHTTP(recB, reqB)
	if recB.Code != http.StatusOK {
		t.Fatalf("user B status = %d body=%s", recB.Code, recB.Body.String())
	}
	var listB gen.TransactionListResponse
	if err := json.NewDecoder(recB.Body).Decode(&listB); err != nil {
		t.Fatalf("decode B: %v", err)
	}
	if listB.Pagination.Total != 0 || len(listB.Data) != 0 {
		t.Fatalf("user B should see 0 transactions, got total=%d len=%d", listB.Pagination.Total, len(listB.Data))
	}

	// Unauthenticated is rejected.
	recUnauth := serve(router, "/api/transactions")
	if recUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("unauth status = %d, want 401", recUnauth.Code)
	}
}

func TestRequireAuth_healthRemainsPublic(t *testing.T) {
	router := chi.NewRouter()
	router.Use(middleware.RequireAuthMiddleware)
	gen.HandlerFromMux(handler.New(noopList{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil), router)

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200", rec.Code)
	}
}
