package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/db"
	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestIntegration_Import(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	t.Cleanup(pool.Close)

	importer := service.NewImport(pool)
	repo := repository.NewTransaction(pool)
	queries := sqldb.New(pool)

	samplePath := filepath.Join("..", "..", "testdata", "sparkasse", "sample.csv")
	overlapPath := filepath.Join("..", "..", "testdata", "sparkasse", "overlap.csv")

	var firstRunID pgtype.UUID

	t.Run("fresh import", func(t *testing.T) {
		result, err := importer.ImportFile(ctx, samplePath, domain.ImportOptions{
			AccountName: "Sparkasse Checking",
		})
		if err != nil {
			t.Fatalf("ImportFile() error = %v", err)
		}
		if result.Rows != 21 {
			t.Fatalf("Rows = %d, want 21", result.Rows)
		}
		if result.Inserted != 21 || result.Duplicates != 0 {
			t.Fatalf("Inserted = %d, Duplicates = %d, want 21/0", result.Inserted, result.Duplicates)
		}
		if result.ImportRunID.String() == "" || result.AccountID.String() == "" {
			t.Fatalf("missing import/account ids: %+v", result)
		}
		if result.AccountName != "Sparkasse Checking" {
			t.Fatalf("AccountName = %q, want Sparkasse Checking", result.AccountName)
		}

		run, err := queries.GetImportRun(ctx, result.ImportRunID)
		if err != nil {
			t.Fatalf("GetImportRun: %v", err)
		}
		if run.Status != domain.ImportRunStatusCommitted {
			t.Fatalf("status = %q, want committed", run.Status)
		}
		if run.RowInserted != 21 || run.RowDuplicate != 0 || run.RowValid != 21 {
			t.Fatalf("run stats = inserted=%d dup=%d valid=%d", run.RowInserted, run.RowDuplicate, run.RowValid)
		}

		linked, err := queries.CountTransactionsByImportRun(ctx, pgtype.UUID{Bytes: result.ImportRunID, Valid: true})
		if err != nil {
			t.Fatalf("CountTransactionsByImportRun: %v", err)
		}
		if linked != 21 {
			t.Fatalf("linked txs = %d, want 21", linked)
		}

		count, err := repo.Count(ctx)
		if err != nil {
			t.Fatalf("Count() error = %v", err)
		}
		if count != 21 {
			t.Fatalf("count = %d, want 21", count)
		}

		firstRunID = pgtype.UUID{Bytes: result.ImportRunID, Valid: true}
	})

	t.Run("re-import same file", func(t *testing.T) {
		result, err := importer.ImportFile(ctx, samplePath, domain.ImportOptions{})
		if err != nil {
			t.Fatalf("ImportFile() error = %v", err)
		}
		if result.Rows != 21 {
			t.Fatalf("Rows = %d, want 21", result.Rows)
		}
		if result.Inserted != 0 || result.Duplicates != 21 {
			t.Fatalf("Inserted = %d, Duplicates = %d, want 0/21", result.Inserted, result.Duplicates)
		}

		run, err := queries.GetImportRun(ctx, result.ImportRunID)
		if err != nil {
			t.Fatalf("GetImportRun: %v", err)
		}
		if run.RowInserted != 0 || run.RowDuplicate != 21 {
			t.Fatalf("run stats = inserted=%d dup=%d", run.RowInserted, run.RowDuplicate)
		}

		// Existing txs keep the first run id; this run inserts nothing.
		linked, err := queries.CountTransactionsByImportRun(ctx, pgtype.UUID{Bytes: result.ImportRunID, Valid: true})
		if err != nil {
			t.Fatalf("CountTransactionsByImportRun: %v", err)
		}
		if linked != 0 {
			t.Fatalf("linked txs for duplicate run = %d, want 0", linked)
		}
		stillFirst, err := queries.CountTransactionsByImportRun(ctx, firstRunID)
		if err != nil {
			t.Fatalf("CountTransactionsByImportRun first: %v", err)
		}
		if stillFirst != 21 {
			t.Fatalf("first run linked = %d, want 21", stillFirst)
		}

		count, err := repo.Count(ctx)
		if err != nil {
			t.Fatalf("Count() error = %v", err)
		}
		if count != 21 {
			t.Fatalf("count = %d, want 21", count)
		}
	})

	t.Run("partial overlap", func(t *testing.T) {
		result, err := importer.ImportFile(ctx, overlapPath, domain.ImportOptions{})
		if err != nil {
			t.Fatalf("ImportFile() error = %v", err)
		}
		if result.Rows != 2 {
			t.Fatalf("Rows = %d, want 2", result.Rows)
		}
		if result.Inserted != 1 || result.Duplicates != 1 {
			t.Fatalf("Inserted = %d, Duplicates = %d, want 1/1", result.Inserted, result.Duplicates)
		}

		count, err := repo.Count(ctx)
		if err != nil {
			t.Fatalf("Count() error = %v", err)
		}
		if count != 22 {
			t.Fatalf("count = %d, want 22", count)
		}
	})

	t.Run("malformed row", func(t *testing.T) {
		malformedPath := filepath.Join(t.TempDir(), "malformed.csv")
		content := readFile(t, samplePath)
		content += `"DE89370400440532013000";"not-a-date";"30.12.25";"KARTENZAHLUNG";"bad row";"";"";"";"";"";"";"";"";"";"-1,00";"EUR";"Umsatz gebucht"` + "\n"
		if err := os.WriteFile(malformedPath, []byte(content), 0o600); err != nil {
			t.Fatalf("write malformed csv: %v", err)
		}

		_, err := importer.ImportFile(ctx, malformedPath, domain.ImportOptions{})
		if err == nil {
			t.Fatal("ImportFile() error = nil, want parse error")
		}

		count, err := repo.Count(ctx)
		if err != nil {
			t.Fatalf("Count() error = %v", err)
		}
		if count != 22 {
			t.Fatalf("count = %d, want 22 (no partial write on parse error)", count)
		}
	})
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

func readFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	return string(data)
}
