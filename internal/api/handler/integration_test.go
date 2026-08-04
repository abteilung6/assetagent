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
	"github.com/google/uuid"
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

	router := newTestRouter(repo, importer)

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

func TestIntegration_ImportsCommitAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	t.Cleanup(pool.Close)

	repo := repository.NewTransaction(pool)
	importer := service.NewImport(pool)
	router := newTestRouter(repo, importer)

	minimalPath := filepath.Join("..", "..", "..", "testdata", "sparkasse", "minimal.csv")
	data, err := os.ReadFile(minimalPath)
	if err != nil {
		t.Fatalf("read minimal: %v", err)
	}
	preview, err := service.PreviewBytes(data, "minimal.csv")
	if err != nil {
		t.Fatalf("PreviewBytes: %v", err)
	}

	t.Run("commit imports transactions", func(t *testing.T) {
		before, err := repo.Count(ctx)
		if err != nil {
			t.Fatalf("Count before: %v", err)
		}

		body, contentType := multipartImport(t, "minimal.csv", data, map[string]string{
			"account_name": "HTTP Sparkasse",
			"preview_hash": preview.FileHash,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/imports", body)
		req.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("commit status = %d body=%s", rec.Code, rec.Body.String())
		}

		var resp gen.ImportCommitResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Inserted != 6 || resp.Duplicates != 0 || resp.Rows != 6 {
			t.Fatalf("commit counts = %+v", resp)
		}
		if resp.AccountName != "HTTP Sparkasse" {
			t.Fatalf("account_name = %q", resp.AccountName)
		}
		if resp.ImportRunId == uuid.Nil {
			t.Fatal("expected import_run_id")
		}

		after, err := repo.Count(ctx)
		if err != nil {
			t.Fatalf("Count after: %v", err)
		}
		if after != before+6 {
			t.Fatalf("transaction count = %d, want %d", after, before+6)
		}

		runs, err := importer.ListRuns(ctx, 10)
		if err != nil {
			t.Fatalf("ListRuns: %v", err)
		}
		if len(runs) == 0 || runs[0].ID != resp.ImportRunId || runs[0].Status != domain.ImportRunStatusCommitted {
			t.Fatalf("import runs = %+v", runs)
		}
	})

	t.Run("recommit reports duplicates", func(t *testing.T) {
		body, contentType := multipartImport(t, "minimal.csv", data, map[string]string{
			"account_name": "HTTP Sparkasse",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/imports", body)
		req.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("recommit status = %d body=%s", rec.Code, rec.Body.String())
		}
		var again gen.ImportCommitResponse
		if err := json.NewDecoder(rec.Body).Decode(&again); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if again.Inserted != 0 || again.Duplicates != 6 {
			t.Fatalf("recommit counts = %+v", again)
		}
	})

	t.Run("preview_hash mismatch", func(t *testing.T) {
		body, contentType := multipartImport(t, "minimal.csv", data, map[string]string{
			"preview_hash": "deadbeef",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/imports", body)
		req.Header.Set("Content-Type", contentType)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		assertValidationFailed(t, rec)
	})
}

func TestIntegration_ImportsLifecycleAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	t.Cleanup(pool.Close)

	repo := repository.NewTransaction(pool)
	importer := service.NewImport(pool)
	router := newTestRouter(repo, importer)

	runAPath := writeTempCSV(t, "run-a.csv", sparkasseHeader()+`
"DE89370400440532013000";"10.01.26";"10.01.26";"KARTENZAHLUNG";"Lifecycle A";"";"";"E2E-LIFE-A";"";"";"";"Cafe A";"DE90100900002868569037";"BEVODEBBXXX";"-10,00";"EUR";"Umsatz gebucht"
`)
	runBPath := writeTempCSV(t, "run-b.csv", sparkasseHeader()+`
"DE89370400440532013000";"11.01.26";"11.01.26";"KARTENZAHLUNG";"Lifecycle B";"";"";"E2E-LIFE-B";"";"";"";"Cafe B";"DE90100900002868569037";"BEVODEBBXXX";"-20,00";"EUR";"Umsatz gebucht"
`)

	resultA, err := importer.ImportFile(ctx, runAPath, domain.ImportOptions{AccountName: "Lifecycle"})
	if err != nil {
		t.Fatalf("ImportFile A: %v", err)
	}
	resultB, err := importer.ImportFile(ctx, runBPath, domain.ImportOptions{AccountName: "Lifecycle"})
	if err != nil {
		t.Fatalf("ImportFile B: %v", err)
	}

	t.Run("list includes both runs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/imports?limit=10", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
		}
		var resp gen.ImportRunListResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(resp.Data) < 2 {
			t.Fatalf("len(data) = %d, want >= 2", len(resp.Data))
		}
		found := map[uuid.UUID]bool{}
		for _, run := range resp.Data {
			found[run.Id] = true
		}
		if !found[resultA.ImportRunID] || !found[resultB.ImportRunID] {
			t.Fatalf("missing runs in list: %+v", resp.Data)
		}
	})

	t.Run("get by id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/imports/"+resultA.ImportRunID.String(), nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
		}
		var run gen.ImportRun
		if err := json.NewDecoder(rec.Body).Decode(&run); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if run.Id != resultA.ImportRunID || run.RowInserted != 1 {
			t.Fatalf("run = %+v", run)
		}
	})

	t.Run("get unknown id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/imports/00000000-0000-0000-0000-000000000099", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404; body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("rollback A leaves B intact", func(t *testing.T) {
		before, err := repo.Count(ctx)
		if err != nil {
			t.Fatalf("Count before: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/imports/"+resultA.ImportRunID.String()+"/rollback", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("rollback status = %d body=%s", rec.Code, rec.Body.String())
		}
		var rb gen.ImportRollbackResponse
		if err := json.NewDecoder(rec.Body).Decode(&rb); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if rb.Deleted != 1 || rb.ImportRunId != resultA.ImportRunID {
			t.Fatalf("rollback = %+v", rb)
		}

		after, err := repo.Count(ctx)
		if err != nil {
			t.Fatalf("Count after: %v", err)
		}
		if after != before-1 {
			t.Fatalf("count = %d, want %d", after, before-1)
		}

		getB := httptest.NewRequest(http.MethodGet, "/api/imports/"+resultB.ImportRunID.String(), nil)
		recB := httptest.NewRecorder()
		router.ServeHTTP(recB, getB)
		if recB.Code != http.StatusOK {
			t.Fatalf("get B status = %d", recB.Code)
		}
		var runB gen.ImportRun
		if err := json.NewDecoder(recB.Body).Decode(&runB); err != nil {
			t.Fatalf("decode B: %v", err)
		}
		if runB.Status != gen.Committed {
			t.Fatalf("B status = %q, want committed", runB.Status)
		}

		getA := httptest.NewRequest(http.MethodGet, "/api/imports/"+resultA.ImportRunID.String(), nil)
		recA := httptest.NewRecorder()
		router.ServeHTTP(recA, getA)
		var runA gen.ImportRun
		if err := json.NewDecoder(recA.Body).Decode(&runA); err != nil {
			t.Fatalf("decode A: %v", err)
		}
		if runA.Status != gen.RolledBack {
			t.Fatalf("A status = %q, want rolled_back", runA.Status)
		}
	})

	t.Run("second rollback conflicts", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/imports/"+resultA.ImportRunID.String()+"/rollback", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409; body=%s", rec.Code, rec.Body.String())
		}
	})
}

func sparkasseHeader() string {
	return `"Auftragskonto";"Buchungstag";"Valutadatum";"Buchungstext";"Verwendungszweck";"Glaeubiger ID";"Mandatsreferenz";"Kundenreferenz (End-to-End)";"Sammlerreferenz";"Lastschrift Ursprungsbetrag";"Auslagenersatz Ruecklastschrift";"Beguenstigter/Zahlungspflichtiger";"Kontonummer/IBAN";"BIC (SWIFT-Code)";"Betrag";"Waehrung";"Info"`
}

func writeTempCSV(t *testing.T, name, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write temp csv: %v", err)
	}
	return path
}

func newTestRouter(repo *repository.Transaction, importer *service.Import) chi.Router {
	router := chi.NewRouter()
	gen.HandlerWithOptions(handler.New(service.NewList(repo), nil, nil, importer, nil, nil, nil, nil, nil, nil, nil, nil, nil), gen.ChiServerOptions{
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
