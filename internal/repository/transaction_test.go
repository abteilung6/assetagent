package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/db"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/shopspring/decimal"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestInsert(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

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
	t.Cleanup(pool.Close)

	repo := repository.NewTransaction(pool)

	if _, err := repo.Insert(ctx, sampleTransaction()); err != nil {
		t.Fatalf("insert: %v", err)
	}

	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestBatchInsert_duplicates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

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
	t.Cleanup(pool.Close)

	repo := repository.NewTransaction(pool)
	transactions := make([]domain.Transaction, 100)
	for i := range transactions {
		transactions[i] = sampleTransactionWithPurpose(fmt.Sprintf("import row %d", i))
	}

	inserted, duplicates, err := repo.BatchInsert(ctx, transactions)
	if err != nil {
		t.Fatalf("batch insert: %v", err)
	}
	if inserted != 100 || duplicates != 0 {
		t.Fatalf("first batch inserted=%d duplicates=%d, want 100/0", inserted, duplicates)
	}

	inserted, duplicates, err = repo.BatchInsert(ctx, transactions)
	if err != nil {
		t.Fatalf("batch insert retry: %v", err)
	}
	if inserted != 0 || duplicates != 100 {
		t.Fatalf("second batch inserted=%d duplicates=%d, want 0/100", inserted, duplicates)
	}

	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 100 {
		t.Fatalf("count = %d, want 100", count)
	}
}

func TestList_paginationAndDateFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

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
	t.Cleanup(pool.Close)

	repo := repository.NewTransaction(pool)

	early := sampleTransactionWithPurpose("early december")
	early.BookingDate = time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	early.ValueDate = early.BookingDate

	late := sampleTransactionWithPurpose("late december")
	late.BookingDate = time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	late.ValueDate = late.BookingDate

	for _, tx := range []domain.Transaction{early, late} {
		if _, err := repo.Insert(ctx, tx); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	from := time.Date(2025, 12, 15, 0, 0, 0, 0, time.UTC)
	result, err := repo.List(ctx, domain.ListParams{
		Limit:    10,
		Offset:   0,
		FromDate: &from,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("total = %d, want 1", result.Total)
	}
	if len(result.Transactions) != 1 {
		t.Fatalf("len(transactions) = %d, want 1", len(result.Transactions))
	}
	if result.Transactions[0].Purpose != "late december" {
		t.Fatalf("purpose = %q, want late december", result.Transactions[0].Purpose)
	}

	page, err := repo.List(ctx, domain.ListParams{Limit: 1, Offset: 1})
	if err != nil {
		t.Fatalf("list page: %v", err)
	}
	if page.Total != 2 {
		t.Fatalf("total = %d, want 2", page.Total)
	}
	if len(page.Transactions) != 1 {
		t.Fatalf("len(transactions) = %d, want 1", len(page.Transactions))
	}
}

func TestList_filters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

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
	t.Cleanup(pool.Close)

	repo := repository.NewTransaction(pool)

	amazon := sampleTransactionWithPurpose("Prime Video subscription")
	amazon.Counterparty = "AMAZON DIGITAL GERMANY GMBH"
	amazon.Amount = decimal.RequireFromString("-2.99")

	netflix := sampleTransactionWithPurpose("Netflix monthly")
	netflix.OrderAccount = "DE89370400440532013000"
	netflix.Counterparty = "NETFLIX INTERNATIONAL B.V."
	netflix.Amount = decimal.RequireFromString("-12.99")
	netflix.BookingText = "SUBSCRIPTION PAYMENT"

	for _, tx := range []domain.Transaction{amazon, netflix} {
		if _, err := repo.Insert(ctx, tx); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	account := "DE15100500006011880043"
	result, err := repo.List(ctx, domain.ListParams{
		Limit:   10,
		Account: &account,
	})
	if err != nil {
		t.Fatalf("list by account: %v", err)
	}
	if result.Total != 1 || result.Transactions[0].Counterparty != amazon.Counterparty {
		t.Fatalf("account filter = %+v, want 1 amazon row", result)
	}

	counterparty := "NETFLIX"
	result, err = repo.List(ctx, domain.ListParams{
		Limit:        10,
		Counterparty: &counterparty,
	})
	if err != nil {
		t.Fatalf("list by counterparty: %v", err)
	}
	if result.Total != 1 || result.Transactions[0].Purpose != "Netflix monthly" {
		t.Fatalf("counterparty filter = %+v", result)
	}

	min := decimal.RequireFromString("-5")
	max := decimal.RequireFromString("0")
	result, err = repo.List(ctx, domain.ListParams{
		Limit:     10,
		MinAmount: &min,
		MaxAmount: &max,
	})
	if err != nil {
		t.Fatalf("list by amount: %v", err)
	}
	if result.Total != 1 || !result.Transactions[0].Amount.Equal(amazon.Amount) {
		t.Fatalf("amount filter = %+v", result)
	}

	search := "subscription"
	result, err = repo.List(ctx, domain.ListParams{
		Limit:  10,
		Search: &search,
	})
	if err != nil {
		t.Fatalf("list by search: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("search total = %d, want 2", result.Total)
	}

	result, err = repo.List(ctx, domain.ListParams{
		Limit:   10,
		Sort:    domain.SortAmount,
		SortAsc: true,
	})
	if err != nil {
		t.Fatalf("list sorted: %v", err)
	}
	if len(result.Transactions) != 2 {
		t.Fatalf("len = %d, want 2", len(result.Transactions))
	}
	if result.Transactions[0].Amount.GreaterThan(result.Transactions[1].Amount) {
		t.Fatalf("expected ascending amount sort, got %s then %s",
			result.Transactions[0].Amount, result.Transactions[1].Amount)
	}
}

func sampleTransactionWithPurpose(purpose string) domain.Transaction {
	tx := sampleTransaction()
	tx.Purpose = purpose
	return tx
}

func sampleTransaction() domain.Transaction {
	counterpartyIBAN := "DE07300308800013011001"
	counterpartyBIC := "TUBDDEDDXXX"

	return domain.Transaction{
		OrderAccount:      "DE15100500006011880043",
		BookingDate:       time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC),
		ValueDate:         time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC),
		BookingText:       "FOLGELASTSCHRIFT",
		Purpose:           "D01-1314859-5720640 Prime Video",
		CreditorID:        "DE96ZZZ00000594888",
		MandateReference:  "ygIP7jsttecE5OCLr)ncS1Y)kw7EQ4",
		EndToEndReference: "78TFZYWYZD7VRM9E",
		Counterparty:      "AMAZON DIGITAL GERMANY GMBH",
		CounterpartyIBAN:  &counterpartyIBAN,
		CounterpartyBIC:   &counterpartyBIC,
		Amount:            decimal.RequireFromString("-2.99"),
		Currency:          "EUR",
		Info:              "Umsatz gebucht",
	}
}
