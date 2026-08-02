package classify_test

import (
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/classify"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestDetectTransferCandidates_exactSameDay(t *testing.T) {
	accA := uuid.New()
	accB := uuid.New()
	outID := uuid.New()
	inID := uuid.New()
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)

	txs := []domain.TransferScanTx{
		{
			ID: outID, AccountID: accA, BookingDate: day,
			Amount: decimal.RequireFromString("-500.00"), BookingText: "UMBUCHUNG",
		},
		{
			ID: inID, AccountID: accB, BookingDate: day,
			Amount: decimal.RequireFromString("500.00"), BookingText: "UMBUCHUNG",
		},
	}

	pairs := classify.DetectTransferCandidates(txs, nil)
	if len(pairs) != 1 {
		t.Fatalf("len = %d, want 1", len(pairs))
	}
	if pairs[0].TxOutID != outID || pairs[0].TxInID != inID {
		t.Fatalf("pair legs = %+v", pairs[0])
	}
	if pairs[0].Confidence != domain.TransferConfidenceExact {
		t.Fatalf("confidence = %q, want exact", pairs[0].Confidence)
	}
	if pairs[0].Status != domain.TransferStatusSuggested {
		t.Fatalf("status = %q", pairs[0].Status)
	}
}

func TestDetectTransferCandidates_skipsSameAccountAndUnrelated(t *testing.T) {
	acc := uuid.New()
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	txs := []domain.TransferScanTx{
		{ID: uuid.New(), AccountID: acc, BookingDate: day, Amount: decimal.RequireFromString("-100.00")},
		{ID: uuid.New(), AccountID: acc, BookingDate: day, Amount: decimal.RequireFromString("100.00")},
		{ID: uuid.New(), AccountID: uuid.New(), BookingDate: day, Amount: decimal.RequireFromString("-40.00")},
		{ID: uuid.New(), AccountID: uuid.New(), BookingDate: day, Amount: decimal.RequireFromString("12.00")},
	}
	if pairs := classify.DetectTransferCandidates(txs, nil); len(pairs) != 0 {
		t.Fatalf("len = %d, want 0", len(pairs))
	}
}

func TestDetectTransferCandidates_skipsExistingLegs(t *testing.T) {
	accA := uuid.New()
	accB := uuid.New()
	outID := uuid.New()
	inID := uuid.New()
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	txs := []domain.TransferScanTx{
		{ID: outID, AccountID: accA, BookingDate: day, Amount: decimal.RequireFromString("-90.00")},
		{ID: inID, AccountID: accB, BookingDate: day, Amount: decimal.RequireFromString("90.00")},
	}
	existing := map[uuid.UUID]struct{}{outID: {}}
	if pairs := classify.DetectTransferCandidates(txs, existing); len(pairs) != 0 {
		t.Fatalf("len = %d, want 0", len(pairs))
	}
}

func TestDetectTransferCandidates_probableWithinWindow(t *testing.T) {
	accA := uuid.New()
	accB := uuid.New()
	day := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	txs := []domain.TransferScanTx{
		{ID: uuid.New(), AccountID: accA, BookingDate: day, Amount: decimal.RequireFromString("-250.00")},
		{ID: uuid.New(), AccountID: accB, BookingDate: day.AddDate(0, 0, 2), Amount: decimal.RequireFromString("250.00")},
	}
	pairs := classify.DetectTransferCandidates(txs, nil)
	if len(pairs) != 1 {
		t.Fatalf("len = %d, want 1", len(pairs))
	}
	if pairs[0].Confidence != domain.TransferConfidenceProbable {
		t.Fatalf("confidence = %q, want probable", pairs[0].Confidence)
	}
}
