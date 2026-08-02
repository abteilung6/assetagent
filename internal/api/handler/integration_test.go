package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/db"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestIntegration_TransactionsAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	t.Cleanup(pool.Close)

	repo := repository.NewTransaction(pool)
	importer := service.NewImport(pool)
	samplePath := filepath.Join("..", "..", "..", "testdata", "sparkasse", "sample.csv")

	result, err := importer.ImportFile(ctx, samplePath, domain.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportFile: %v", err)
	}
	if result.Inserted != 21 {
		t.Fatalf("inserted = %d, want 21", result.Inserted)
	}

	router := newTestRouter(repo)

	t.Run("paginated list", func(t *testing.T) {
		rec := serve(router, "/api/transactions?limit=5")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}

		var resp gen.TransactionListResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Data) != 5 {
			t.Fatalf("len(data) = %d, want 5", len(resp.Data))
		}
		if resp.Pagination.Limit != 5 || resp.Pagination.Total != 21 {
			t.Fatalf("pagination = %+v, want limit=5 total=21", resp.Pagination)
		}
		if resp.Data[0].Amount == "" || resp.Data[0].OrderAccount == "" {
			t.Fatalf("transaction fields not populated: %+v", resp.Data[0])
		}
	})

	t.Run("date filter shrinks total", func(t *testing.T) {
		rec := serve(router, "/api/transactions?from=2025-12-30&to=2025-12-30")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
		}

		var resp gen.TransactionListResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Pagination.Total != 4 {
			t.Fatalf("total = %d, want 4", resp.Pagination.Total)
		}
	})

	t.Run("invalid limit zero", func(t *testing.T) {
		rec := serve(router, "/api/transactions?limit=0")
		assertValidationFailed(t, rec)
	})

	t.Run("invalid limit above max", func(t *testing.T) {
		rec := serve(router, "/api/transactions?limit=999")
		assertValidationFailed(t, rec)
	})

	t.Run("malformed date", func(t *testing.T) {
		rec := serve(router, "/api/transactions?from=not-a-date")
		assertValidationFailed(t, rec)
	})

	t.Run("preview does not write transactions", func(t *testing.T) {
		before, err := repo.Count(ctx)
		if err != nil {
			t.Fatalf("Count before: %v", err)
		}

		minimalPath := filepath.Join("..", "..", "..", "testdata", "sparkasse", "minimal.csv")
		data, err := os.ReadFile(minimalPath)
		if err != nil {
			t.Fatalf("read minimal: %v", err)
		}
		body, contentType := multipartFile(t, "minimal.csv", data)
		req := httptest.NewRequest(http.MethodPost, "/api/imports/preview", body)
		req.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("preview status = %d body=%s", rec.Code, rec.Body.String())
		}

		after, err := repo.Count(ctx)
		if err != nil {
			t.Fatalf("Count after: %v", err)
		}
		if after != before {
			t.Fatalf("transaction count changed: before=%d after=%d", before, after)
		}
	})
}

func newTestRouter(repo *repository.Transaction) chi.Router {
	router := chi.NewRouter()
	gen.HandlerWithOptions(handler.New(service.NewList(repo), nil, nil), gen.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handler.APIErrorHandler,
	})
	return router
}

func serve(router chi.Router, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func assertValidationFailed(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}

	var resp gen.Error
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Error != "validation_failed" {
		t.Fatalf("error = %q, want validation_failed", resp.Error)
	}
	if resp.Message == "" {
		t.Fatal("expected non-empty error message")
	}
}

func setupPostgres(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	container, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("assetagent"),
		postgres.WithUsername("assetagent"),
		postgres.WithPassword("assetagent"),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("terminate postgres: %v", err)
		}
	})

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	waitForDB(ctx, t, connStr)

	if err := db.RunMigrations(connStr, "up"); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	pool, err := db.NewPool(ctx, connStr)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}

	return pool
}

func waitForDB(ctx context.Context, t *testing.T, connStr string) {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		pool, err := db.NewPool(ctx, connStr)
		if err == nil {
			if err := pool.Ping(ctx); err == nil {
				pool.Close()
				return
			}
			pool.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Fatal("database not ready")
}
