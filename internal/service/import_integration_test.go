package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/db"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/abteilung6/assetagent/internal/service"
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

	importer := service.NewImport(repository.NewTransaction(pool))
	repo := repository.NewTransaction(pool)

	samplePath := filepath.Join("..", "..", "testdata", "sparkasse", "sample.csv")
	overlapPath := filepath.Join("..", "..", "testdata", "sparkasse", "overlap.csv")

	t.Run("fresh import", func(t *testing.T) {
		result, err := importer.ImportFile(ctx, samplePath)
		if err != nil {
			t.Fatalf("ImportFile() error = %v", err)
		}
		if result.Rows != 21 {
			t.Fatalf("Rows = %d, want 21", result.Rows)
		}
		if result.Inserted != 21 || result.Duplicates != 0 {
			t.Fatalf("Inserted = %d, Duplicates = %d, want 21/0", result.Inserted, result.Duplicates)
		}

		count, err := repo.Count(ctx)
		if err != nil {
			t.Fatalf("Count() error = %v", err)
		}
		if count != 21 {
			t.Fatalf("count = %d, want 21", count)
		}
	})

	t.Run("re-import same file", func(t *testing.T) {
		result, err := importer.ImportFile(ctx, samplePath)
		if err != nil {
			t.Fatalf("ImportFile() error = %v", err)
		}
		if result.Rows != 21 {
			t.Fatalf("Rows = %d, want 21", result.Rows)
		}
		if result.Inserted != 0 || result.Duplicates != 21 {
			t.Fatalf("Inserted = %d, Duplicates = %d, want 0/21", result.Inserted, result.Duplicates)
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
		result, err := importer.ImportFile(ctx, overlapPath)
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

		_, err := importer.ImportFile(ctx, malformedPath)
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
