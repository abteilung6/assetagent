package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
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

func TestIntegration_TransferConfirmCashflowV2(t *testing.T) {
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
	insertTransferTx(t, ctx, q, checking.ID, "DE111", day, "-500.00", "UMBUCHUNG", "fp-out-cf")
	insertTransferTx(t, ctx, q, savings.ID, "DE222", day, "500.00", "UMBUCHUNG", "fp-in-cf")
	insertTransferTx(t, ctx, q, checking.ID, "DE111", day, "-12.50", "EDEKA", "fp-grocery-cf")
	insertTransferTx(t, ctx, q, checking.ID, "DE111", day, "2000.00", "GEHALT", "fp-salary-cf")

	transfers := service.NewTransfers(pool)
	scan, err := transfers.Scan(ctx)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if scan.Suggested != 1 {
		t.Fatalf("suggested = %d, want 1", scan.Suggested)
	}

	reports := repository.NewReports(pool)
	from := day
	to := day

	raw, err := reports.GetCashflow(ctx, from, to)
	if err != nil {
		t.Fatalf("raw cashflow: %v", err)
	}
	if !raw.Income.Equal(decimal.RequireFromString("2500.00")) {
		t.Fatalf("raw income = %s, want 2500.00", raw.Income)
	}
	if !raw.Expenses.Equal(decimal.RequireFromString("512.50")) {
		t.Fatalf("raw expenses = %s, want 512.50", raw.Expenses)
	}

	before, err := reports.GetCashflowV2(ctx, from, to)
	if err != nil {
		t.Fatalf("cashflow v2 before confirm: %v", err)
	}
	if !before.Income.Equal(raw.Income) || !before.Expenses.Equal(raw.Expenses) {
		t.Fatalf("v2 before confirm should match raw; got income=%s expenses=%s", before.Income, before.Expenses)
	}

	confirmed, err := transfers.Confirm(ctx, scan.Pairs[0].ID)
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if confirmed.Status != domain.TransferStatusConfirmed {
		t.Fatalf("status = %q, want confirmed", confirmed.Status)
	}

	after, err := reports.GetCashflowV2(ctx, from, to)
	if err != nil {
		t.Fatalf("cashflow v2 after confirm: %v", err)
	}
	if !after.Income.Equal(decimal.RequireFromString("2000.00")) {
		t.Fatalf("v2 income = %s, want 2000.00 (transfer in excluded)", after.Income)
	}
	if !after.Expenses.Equal(decimal.RequireFromString("12.50")) {
		t.Fatalf("v2 expenses = %s, want 12.50 (transfer out excluded)", after.Expenses)
	}
	if !after.Net.Equal(decimal.RequireFromString("1987.50")) {
		t.Fatalf("v2 net = %s, want 1987.50", after.Net)
	}
	if !after.TransfersExcluded {
		t.Fatal("TransfersExcluded = false, want true")
	}

	if _, err := transfers.Confirm(ctx, scan.Pairs[0].ID); !errors.Is(err, service.ErrTransferPairNotSuggested) {
		t.Fatalf("double confirm err = %v, want ErrTransferPairNotSuggested", err)
	}

	// Reject path on a fresh suggestion
	out2 := insertTransferTx(t, ctx, q, checking.ID, "DE111", day.AddDate(0, 0, 1), "-80.00", "UMBUCHUNG", "fp-out-rej")
	in2 := insertTransferTx(t, ctx, q, savings.ID, "DE222", day.AddDate(0, 0, 1), "80.00", "UMBUCHUNG", "fp-in-rej")
	_ = out2
	_ = in2
	scan2, err := transfers.Scan(ctx)
	if err != nil {
		t.Fatalf("scan2: %v", err)
	}
	if scan2.Suggested != 1 {
		t.Fatalf("scan2 suggested = %d, want 1", scan2.Suggested)
	}
	rejected, err := transfers.Reject(ctx, scan2.Pairs[0].ID)
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if rejected.Status != domain.TransferStatusRejected {
		t.Fatalf("reject status = %q", rejected.Status)
	}
	still, err := reports.GetCashflowV2(ctx, day, day.AddDate(0, 0, 1))
	if err != nil {
		t.Fatalf("cashflow after reject: %v", err)
	}
	// rejected 80 pair still counts; confirmed 500 pair still excluded
	if !still.Expenses.Equal(decimal.RequireFromString("92.50")) {
		t.Fatalf("expenses after reject = %s, want 92.50", still.Expenses)
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
