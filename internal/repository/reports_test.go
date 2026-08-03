package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/db"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestReports_GetCashflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := setupReportsDB(ctx, t)
	t.Cleanup(pool.Close)

	txRepo := repository.NewTransaction(pool)
	reports := repository.NewReports(pool)

	salary := sampleTransactionWithPurpose("December salary")
	salary.BookingDate = time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC)
	salary.ValueDate = salary.BookingDate
	salary.Counterparty = "Example Employer GmbH"
	salary.Amount = decimal.RequireFromString("56000.00")

	amazon := sampleTransactionWithPurpose("Prime Video")
	amazon.BookingDate = time.Date(2025, 12, 20, 0, 0, 0, 0, time.UTC)
	amazon.ValueDate = amazon.BookingDate
	amazon.Amount = decimal.RequireFromString("-2.99")

	netflix := sampleTransactionWithPurpose("Streaming")
	netflix.BookingDate = time.Date(2025, 12, 10, 0, 0, 0, 0, time.UTC)
	netflix.ValueDate = netflix.BookingDate
	netflix.Counterparty = "NETFLIX INTERNATIONAL B.V."
	netflix.Amount = decimal.RequireFromString("-12.99")

	outOfRange := sampleTransactionWithPurpose("January rent")
	outOfRange.BookingDate = time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	outOfRange.ValueDate = outOfRange.BookingDate
	outOfRange.Amount = decimal.RequireFromString("-1200.00")

	for _, tx := range []domain.Transaction{salary, amazon, netflix, outOfRange} {
		if _, err := txRepo.Insert(ctx, tx); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	from := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	report, err := reports.GetCashflow(ctx, from, to)
	if err != nil {
		t.Fatalf("get cashflow: %v", err)
	}

	assertDecimalEqual(t, report.Income, "56000.00")
	assertDecimalEqual(t, report.Expenses, "15.98")
	assertDecimalEqual(t, report.Net, "55984.02")
}

func TestReports_GetCashflowV2_excludesOneOff(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := setupReportsDB(ctx, t)
	t.Cleanup(pool.Close)

	txRepo := repository.NewTransaction(pool)
	reports := repository.NewReports(pool)

	regular := sampleTransactionWithPurpose("Groceries")
	regular.BookingDate = time.Date(2025, 12, 12, 0, 0, 0, 0, time.UTC)
	regular.ValueDate = regular.BookingDate
	regular.Amount = decimal.RequireFromString("-100.00")

	oneOff := sampleTransactionWithPurpose("Down payment")
	oneOff.BookingDate = time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC)
	oneOff.ValueDate = oneOff.BookingDate
	oneOff.Amount = decimal.RequireFromString("-50000.00")

	regularID, err := txRepo.Insert(ctx, regular)
	if err != nil {
		t.Fatalf("insert regular: %v", err)
	}
	oneOffID, err := txRepo.Insert(ctx, oneOff)
	if err != nil {
		t.Fatalf("insert one-off: %v", err)
	}
	if _, err := txRepo.SetOneOff(ctx, oneOffID, true); err != nil {
		t.Fatalf("set one-off: %v", err)
	}

	from := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	report, err := reports.GetCashflowV2(ctx, from, to)
	if err != nil {
		t.Fatalf("get cashflow v2: %v", err)
	}
	assertDecimalEqual(t, report.Expenses, "100.00")

	evidence, err := reports.GetCashflowV2Evidence(ctx, from, to)
	if err != nil {
		t.Fatalf("get cashflow v2 evidence: %v", err)
	}
	wantRegular := "tx_" + regularID.String()
	wantOneOff := "tx_" + oneOffID.String()
	foundRegular := false
	for _, id := range evidence.EvidenceIDs {
		if id == wantOneOff {
			t.Fatalf("one-off transaction %s should not appear in evidence", oneOffID)
		}
		if id == wantRegular {
			foundRegular = true
		}
	}
	if !foundRegular {
		t.Fatalf("expected regular transaction %s in evidence", regularID)
	}

	months, err := reports.ListMonthlyCashflowV2(ctx, from, to)
	if err != nil {
		t.Fatalf("list monthly cashflow: %v", err)
	}
	if len(months) != 1 {
		t.Fatalf("months = %d, want 1", len(months))
	}
	assertDecimalEqual(t, months[0].Expenses, "100.00")
}

func TestReports_GetTopCounterparties(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := setupReportsDB(ctx, t)
	t.Cleanup(pool.Close)

	txRepo := repository.NewTransaction(pool)
	reports := repository.NewReports(pool)

	amazon := sampleTransactionWithPurpose("Prime Video")
	amazon.BookingDate = time.Date(2025, 12, 20, 0, 0, 0, 0, time.UTC)
	amazon.ValueDate = amazon.BookingDate
	amazon.Amount = decimal.RequireFromString("-2.99")

	netflix := sampleTransactionWithPurpose("Streaming")
	netflix.BookingDate = time.Date(2025, 12, 10, 0, 0, 0, 0, time.UTC)
	netflix.ValueDate = netflix.BookingDate
	netflix.Counterparty = "NETFLIX INTERNATIONAL B.V."
	netflix.Amount = decimal.RequireFromString("-12.99")

	fee := sampleTransactionWithPurpose("Monthly fee")
	fee.BookingDate = time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	fee.ValueDate = fee.BookingDate
	fee.Counterparty = ""
	fee.Amount = decimal.RequireFromString("-4.95")

	for _, tx := range []domain.Transaction{amazon, netflix, fee} {
		if _, err := txRepo.Insert(ctx, tx); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	from := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	spends, err := reports.GetTopCounterparties(ctx, from, to, 5)
	if err != nil {
		t.Fatalf("get top counterparties: %v", err)
	}
	if len(spends) != 2 {
		t.Fatalf("len(spends) = %d, want 2", len(spends))
	}
	if spends[0].Counterparty != "NETFLIX INTERNATIONAL B.V." {
		t.Fatalf("top counterparty = %q, want NETFLIX", spends[0].Counterparty)
	}
	assertDecimalEqual(t, spends[0].TotalSpent, "12.99")
	if spends[0].TransactionCount != 1 {
		t.Fatalf("transaction count = %d, want 1", spends[0].TransactionCount)
	}
	if spends[1].Counterparty != "AMAZON DIGITAL GERMANY GMBH" {
		t.Fatalf("second counterparty = %q, want AMAZON", spends[1].Counterparty)
	}
	assertDecimalEqual(t, spends[1].TotalSpent, "2.99")
}

func setupReportsDB(ctx context.Context, t *testing.T) *pgxpool.Pool {
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

func assertDecimalEqual(t *testing.T, got decimal.Decimal, want string) {
	t.Helper()

	expected := decimal.RequireFromString(want)
	if !got.Equal(expected) {
		t.Fatalf("decimal = %s, want %s", got, expected)
	}
}
