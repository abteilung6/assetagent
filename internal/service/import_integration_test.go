package service_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/db"
	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
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

	t.Run("import_flow fixture", func(t *testing.T) {
		flowPath := filepath.Join("..", "..", "testdata", "sparkasse", "import_flow.csv")
		result, err := importer.ImportFile(ctx, flowPath, domain.ImportOptions{
			AccountName: "Flow Account",
		})
		if err != nil {
			t.Fatalf("ImportFile() error = %v", err)
		}
		if result.Inserted != 2 || result.Duplicates != 0 {
			t.Fatalf("Inserted = %d, Duplicates = %d, want 2/0", result.Inserted, result.Duplicates)
		}

		again, err := importer.ImportFile(ctx, flowPath, domain.ImportOptions{})
		if err != nil {
			t.Fatalf("reimport error = %v", err)
		}
		if again.Inserted != 0 || again.Duplicates != 2 {
			t.Fatalf("reimport Inserted = %d, Duplicates = %d, want 0/2", again.Inserted, again.Duplicates)
		}
	})

	t.Run("latin1 fixture commits", func(t *testing.T) {
		latin1Path := filepath.Join("..", "..", "testdata", "sparkasse", "latin1.csv")
		result, err := importer.ImportFile(ctx, latin1Path, domain.ImportOptions{
			AccountName: "Latin1 Account",
		})
		if err != nil {
			t.Fatalf("ImportFile() error = %v", err)
		}
		if result.Inserted != 2 {
			t.Fatalf("Inserted = %d, want 2", result.Inserted)
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
		if count != 26 {
			t.Fatalf("count = %d, want 26 (no partial write on parse error)", count)
		}
	})
}

func TestIntegration_ImportRollback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	t.Cleanup(pool.Close)

	importer := service.NewImport(pool)
	repo := repository.NewTransaction(pool)
	reports := repository.NewReports(pool)
	queries := sqldb.New(pool)

	minimalPath := filepath.Join("..", "..", "testdata", "sparkasse", "minimal.csv")
	runAPath := filepath.Join(t.TempDir(), "run-a.csv")
	if err := os.WriteFile(runAPath, []byte(readFile(t, minimalPath)), 0o600); err != nil {
		t.Fatalf("write run-a: %v", err)
	}

	runBPath := filepath.Join(t.TempDir(), "run-b.csv")
	runBCSV := sparkasseHeader() + "\n" +
		`"DE89370400440532013000";"15.01.26";"15.01.26";"KARTENZAHLUNG";"Unique rollback B row";"";"";"E2E-ROLLBACK-B";"";"";"";"Rollback Cafe";"DE90100900002868569037";"BEVODEBBXXX";"-42,00";"EUR";"Umsatz gebucht"` + "\n"
	if err := os.WriteFile(runBPath, []byte(runBCSV), 0o600); err != nil {
		t.Fatalf("write run-b: %v", err)
	}

	resultA, err := importer.ImportFile(ctx, runAPath, domain.ImportOptions{AccountName: "Rollback Test"})
	if err != nil {
		t.Fatalf("import A: %v", err)
	}
	if resultA.Inserted != 6 {
		t.Fatalf("A inserted = %d, want 6", resultA.Inserted)
	}

	resultB, err := importer.ImportFile(ctx, runBPath, domain.ImportOptions{})
	if err != nil {
		t.Fatalf("import B: %v", err)
	}
	if resultB.Inserted != 1 {
		t.Fatalf("B inserted = %d, want 1", resultB.Inserted)
	}

	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 7 {
		t.Fatalf("count before rollback = %d, want 7", count)
	}

	rollback, err := importer.Rollback(ctx, resultA.ImportRunID)
	if err != nil {
		t.Fatalf("Rollback A: %v", err)
	}
	if rollback.Deleted != 6 {
		t.Fatalf("deleted = %d, want 6", rollback.Deleted)
	}

	runA, err := queries.GetImportRun(ctx, resultA.ImportRunID)
	if err != nil {
		t.Fatalf("GetImportRun A: %v", err)
	}
	if runA.Status != domain.ImportRunStatusRolledBack {
		t.Fatalf("A status = %q, want rolled_back", runA.Status)
	}

	linkedA, err := queries.CountTransactionsByImportRun(ctx, pgtype.UUID{Bytes: resultA.ImportRunID, Valid: true})
	if err != nil {
		t.Fatalf("linked A: %v", err)
	}
	if linkedA != 0 {
		t.Fatalf("linked A = %d, want 0", linkedA)
	}

	linkedB, err := queries.CountTransactionsByImportRun(ctx, pgtype.UUID{Bytes: resultB.ImportRunID, Valid: true})
	if err != nil {
		t.Fatalf("linked B: %v", err)
	}
	if linkedB != 1 {
		t.Fatalf("linked B = %d, want 1", linkedB)
	}

	count, err = repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count after rollback: %v", err)
	}
	if count != 1 {
		t.Fatalf("count after rollback = %d, want 1", count)
	}

	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	cashflow, err := reports.GetCashflow(ctx, from, to)
	if err != nil {
		t.Fatalf("GetCashflow: %v", err)
	}
	if !cashflow.Expenses.Equal(mustDecimal(t, "42.00")) {
		t.Fatalf("expenses = %s, want 42.00", cashflow.Expenses)
	}

	if _, err := importer.Rollback(ctx, resultA.ImportRunID); err == nil {
		t.Fatal("second Rollback A error = nil, want already rolled back")
	} else if !errors.Is(err, service.ErrImportRunAlreadyRolledBack) {
		t.Fatalf("second Rollback A error = %v, want ErrImportRunAlreadyRolledBack", err)
	}

	if _, err := importer.Rollback(ctx, uuid.MustParse("00000000-0000-0000-0000-000000000099")); err == nil {
		t.Fatal("unknown Rollback error = nil, want not found")
	} else if !errors.Is(err, service.ErrImportRunNotFound) {
		t.Fatalf("unknown Rollback error = %v, want ErrImportRunNotFound", err)
	}

	runs, err := importer.ListRuns(ctx, 10)
	if err != nil {
		t.Fatalf("ListRuns: %v", err)
	}
	if len(runs) < 2 {
		t.Fatalf("ListRuns len = %d, want >= 2", len(runs))
	}
}

func sparkasseHeader() string {
	return `"Auftragskonto";"Buchungstag";"Valutadatum";"Buchungstext";"Verwendungszweck";"Glaeubiger ID";"Mandatsreferenz";"Kundenreferenz (End-to-End)";"Sammlerreferenz";"Lastschrift Ursprungsbetrag";"Auslagenersatz Ruecklastschrift";"Beguenstigter/Zahlungspflichtiger";"Kontonummer/IBAN";"BIC (SWIFT-Code)";"Betrag";"Waehrung";"Info"`
}

func mustDecimal(t *testing.T, raw string) decimal.Decimal {
	t.Helper()
	value, err := decimal.NewFromString(raw)
	if err != nil {
		t.Fatalf("decimal %q: %v", raw, err)
	}
	return value
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
