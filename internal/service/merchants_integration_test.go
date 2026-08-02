package service_test

import (
	"context"
	"testing"
	"time"

	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

func TestIntegration_MerchantRebuild(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	t.Cleanup(pool.Close)

	q := sqldb.New(pool)
	acc, err := q.CreateAccount(ctx, sqldb.CreateAccountParams{
		DisplayName:      "Checking",
		Bank:             "sparkasse",
		Currency:         "EUR",
		OrderAccount:     pgtype.Text{String: "DE-M-1", Valid: true},
		MaskedIdentifier: "DE…",
	})
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	insertMerchantTx(t, ctx, q, acc.ID, "AMAZON DIGITAL GERMANY GMBH", "Prime", "fp-amz-1", day)
	insertMerchantTx(t, ctx, q, acc.ID, "Amazon Payments Europe S.C.A.", "Order", "fp-amz-2", day)
	insertMerchantTx(t, ctx, q, acc.ID, "PayPal Europe S.a.r.l. et Cie S.C.A", "Shop", "fp-pp-1", day)

	svc := service.NewMerchants(pool)
	first, err := svc.Rebuild(ctx)
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if first.MerchantsCreated != 2 || first.AliasesCreated != 2 {
		t.Fatalf("first = %+v, want 2 merchants/aliases (amazon collapsed)", first)
	}

	second, err := svc.Rebuild(ctx)
	if err != nil {
		t.Fatalf("rebuild 2: %v", err)
	}
	if second.MerchantsCreated != 0 || second.AliasesCreated != 0 {
		t.Fatalf("second not idempotent: %+v", second)
	}
	if second.AliasesExisting != 2 {
		t.Fatalf("existing aliases = %d, want 2", second.AliasesExisting)
	}

	listed, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("list len = %d, want 2", len(listed))
	}
}

func insertMerchantTx(
	t *testing.T,
	ctx context.Context,
	q *sqldb.Queries,
	accountID uuid.UUID,
	counterparty, purpose, fingerprint string,
	day time.Time,
) {
	t.Helper()
	_, err := q.InsertTransaction(ctx, sqldb.InsertTransactionParams{
		OrderAccount: "DE-M-1",
		BookingDate:  pgtype.Date{Time: day, Valid: true},
		ValueDate:    pgtype.Date{Time: day, Valid: true},
		BookingText:  "LASTSCHRIFT",
		Purpose:      purpose,
		Counterparty: counterparty,
		Amount:       decimal.RequireFromString("-1.00"),
		Currency:     "EUR",
		Fingerprint:  fingerprint,
		AccountID:    pgtype.UUID{Bytes: accountID, Valid: true},
	})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
}
