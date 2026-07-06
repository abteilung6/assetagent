package repository_test

import (
	"context"
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
