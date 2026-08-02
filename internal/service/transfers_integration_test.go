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

func TestIntegration_TransferScan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	t.Cleanup(pool.Close)

	q := sqldb.New(pool)
	checking, err := q.CreateAccount(ctx, sqldb.CreateAccountParams{
		DisplayName:      "Checking",
		Bank:             "sparkasse",
		Currency:         "EUR",
		OrderAccount:     pgtype.Text{String: "DE111", Valid: true},
		MaskedIdentifier: "DE11…",
	})
	if err != nil {
		t.Fatalf("create checking: %v", err)
	}
	savings, err := q.CreateAccount(ctx, sqldb.CreateAccountParams{
		DisplayName:      "Savings",
		Bank:             "sparkasse",
		Currency:         "EUR",
		OrderAccount:     pgtype.Text{String: "DE222", Valid: true},
		MaskedIdentifier: "DE22…",
	})
	if err != nil {
		t.Fatalf("create savings: %v", err)
	}

	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	outID := insertTransferTx(t, ctx, q, checking.ID, "DE111", day, "-500.00", "UMBUCHUNG", "fp-out")
	inID := insertTransferTx(t, ctx, q, savings.ID, "DE222", day, "500.00", "UMBUCHUNG", "fp-in")
	insertTransferTx(t, ctx, q, checking.ID, "DE111", day, "-12.50", "EDEKA", "fp-grocery")

	svc := service.NewTransfers(pool)
	first, err := svc.Scan(ctx)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if first.Suggested != 1 {
		t.Fatalf("suggested = %d, want 1", first.Suggested)
	}
	if len(first.Pairs) != 1 {
		t.Fatalf("pairs = %d, want 1", len(first.Pairs))
	}
	pair := first.Pairs[0]
	if pair.TxOutID != outID || pair.TxInID != inID {
		t.Fatalf("legs out=%s in=%s, want %s / %s", pair.TxOutID, pair.TxInID, outID, inID)
	}
	if pair.Confidence != domain.TransferConfidenceExact {
		t.Fatalf("confidence = %q, want exact", pair.Confidence)
	}
	if pair.Status != domain.TransferStatusSuggested {
		t.Fatalf("status = %q", pair.Status)
	}

	second, err := svc.Scan(ctx)
	if err != nil {
		t.Fatalf("rescan: %v", err)
	}
	if second.Suggested != 0 {
		t.Fatalf("rescan suggested = %d, want 0", second.Suggested)
	}

	listed, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("list len = %d, want 1", len(listed))
	}
}

func insertTransferTx(
	t *testing.T,
	ctx context.Context,
	q *sqldb.Queries,
	accountID uuid.UUID,
	orderAccount string,
	day time.Time,
	amount string,
	bookingText string,
	fingerprint string,
) uuid.UUID {
	t.Helper()
	id, err := q.InsertTransaction(ctx, sqldb.InsertTransactionParams{
		OrderAccount: orderAccount,
		BookingDate:  pgtype.Date{Time: day, Valid: true},
		ValueDate:    pgtype.Date{Time: day, Valid: true},
		BookingText:  bookingText,
		Purpose:      bookingText,
		Amount:       decimal.RequireFromString(amount),
		Currency:     "EUR",
		Fingerprint:  fingerprint,
		AccountID:    pgtype.UUID{Bytes: accountID, Valid: true},
	})
	if err != nil {
		t.Fatalf("insert tx %s: %v", fingerprint, err)
	}
	return id
}
