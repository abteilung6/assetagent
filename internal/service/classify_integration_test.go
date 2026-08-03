package service_test

import (
	"context"
	"testing"
	"time"

	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

func TestIntegration_ClassifyRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	t.Cleanup(pool.Close)

	q := sqldb.New(pool)
	checking, err := q.CreateAccount(ctx, sqldb.CreateAccountParams{
		DisplayName: "Checking", Bank: "sparkasse", Currency: "EUR",
		OrderAccount: pgtype.Text{String: "DE-C-1", Valid: true}, MaskedIdentifier: "A",
	})
	if err != nil {
		t.Fatalf("checking: %v", err)
	}
	savings, err := q.CreateAccount(ctx, sqldb.CreateAccountParams{
		DisplayName: "Savings", Bank: "sparkasse", Currency: "EUR",
		OrderAccount: pgtype.Text{String: "DE-S-1", Valid: true}, MaskedIdentifier: "B",
	})
	if err != nil {
		t.Fatalf("savings: %v", err)
	}

	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	outID := insertClassifyTx(t, ctx, q, checking.ID, "DE-C-1", "-200.00", "UMBUCHUNG", "UMBUCHUNG", "fp-c-out", day)
	inID := insertClassifyTx(t, ctx, q, savings.ID, "DE-S-1", "200.00", "UMBUCHUNG", "UMBUCHUNG", "fp-c-in", day)
	insertClassifyTx(t, ctx, q, checking.ID, "DE-C-1", "-23.50", "REWE Dortmund", "Einkauf", "fp-c-rewe", day)
	insertClassifyTx(t, ctx, q, checking.ID, "DE-C-1", "2500.00", "Example Employer GmbH", "Gehalt", "fp-c-salary", day)

	_, err = q.InsertTransferPair(ctx, sqldb.InsertTransferPairParams{
		TxOutID: outID, TxInID: inID,
		Status: domain.TransferStatusSuggested, Confidence: domain.TransferConfidenceExact,
		Rationale: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("pair: %v", err)
	}
	if _, err := service.NewTransfers(pool).Confirm(ctx, mustPairID(t, ctx, q)); err != nil {
		t.Fatalf("confirm: %v", err)
	}

	svc := service.NewClassify(pool)
	if _, err := svc.ImportPatternRulesCSV(ctx, "../../testdata/classification_patterns.csv"); err != nil {
		t.Fatalf("import patterns: %v", err)
	}
	first, err := svc.Run(ctx)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if first.Upserted != 4 {
		t.Fatalf("upserted = %d, want 4", first.Upserted)
	}
	if first.ByCategory["transfer"] != 2 {
		t.Fatalf("transfer count = %d, want 2; by=%v", first.ByCategory["transfer"], first.ByCategory)
	}
	if first.ByCategory["groceries"] != 1 {
		t.Fatalf("groceries = %d, by=%v", first.ByCategory["groceries"], first.ByCategory)
	}
	if first.ByCategory["income"] != 1 {
		t.Fatalf("income = %d, by=%v", first.ByCategory["income"], first.ByCategory)
	}

	second, err := svc.Run(ctx)
	if err != nil {
		t.Fatalf("rerun: %v", err)
	}
	if second.Upserted != 4 || second.SkippedUser != 0 {
		t.Fatalf("rerun = %+v", second)
	}
}

func insertClassifyTx(
	t *testing.T,
	ctx context.Context,
	q *sqldb.Queries,
	accountID uuid.UUID,
	order, amount, counterparty, purpose, fp string,
	day time.Time,
) uuid.UUID {
	t.Helper()
	id, err := q.InsertTransaction(ctx, sqldb.InsertTransactionParams{
		OrderAccount: order,
		BookingDate:  pgtype.Date{Time: day, Valid: true},
		ValueDate:    pgtype.Date{Time: day, Valid: true},
		BookingText:  "TEST",
		Purpose:      purpose,
		Counterparty: counterparty,
		Amount:       decimal.RequireFromString(amount),
		Currency:     "EUR",
		Fingerprint:  fp,
		AccountID:    pgtype.UUID{Bytes: accountID, Valid: true},
	})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	return id
}

func mustPairID(t *testing.T, ctx context.Context, q *sqldb.Queries) uuid.UUID {
	t.Helper()
	pairs, err := q.ListTransferPairs(ctx)
	if err != nil || len(pairs) == 0 {
		t.Fatalf("list pairs: %v len=%d", err, len(pairs))
	}
	return pairs[0].ID
}
