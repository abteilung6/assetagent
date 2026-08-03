package review_test

import (
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/review"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestGenerate_capsAtThreeAndPrioritizes(t *testing.T) {
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	s1 := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	s2 := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	tx := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	summary, findings := review.Generate(review.Input{
		PeriodFrom:              from,
		PeriodTo:                to,
		SustainableFreeCashflow: decimal.RequireFromString("-200.00"),
		Series: []domain.RecurringSeries{
			{
				ID: s1, DisplayName: "Netflix", Status: domain.RecurringStatusActive,
				AmountChanged: true, AmountTypical: decimal.RequireFromString("15.99"),
				AmountLast: decimal.RequireFromString("17.99"),
			},
			{
				ID: s2, DisplayName: "Gym", Status: domain.RecurringStatusUncertain,
				AmountTypical: decimal.RequireFromString("40.00"), AmountLast: decimal.RequireFromString("40.00"),
			},
		},
		LargeExpenses: []review.LargeExpense{{
			TransactionID: tx.String(),
			Label:         "IKEA",
			Amount:        decimal.RequireFromString("800.00"),
		}},
		NeedsReviewCount: 4,
	})

	if len(findings) != 3 {
		t.Fatalf("len = %d, want 3", len(findings))
	}
	if findings[0].Type != review.FindingFreeCashflowPressure {
		t.Fatalf("first = %q", findings[0].Type)
	}
	if findings[1].Type != review.FindingRecurringAmountChange {
		t.Fatalf("second = %q", findings[1].Type)
	}
	if findings[2].Type != review.FindingLargeExpense {
		t.Fatalf("third = %q", findings[2].Type)
	}
	if summary == "" {
		t.Fatal("empty summary")
	}
	// Uncertain + queue residue dropped by cap.
	for _, f := range findings {
		if f.Type == review.FindingUncertainRecurring || f.Type == review.FindingNeedsReviewResidue {
			t.Fatalf("unexpected type %q under cap", f.Type)
		}
		if f.Amount == nil && f.Type != review.FindingNeedsReviewResidue {
			// free cashflow and others should have amounts
		}
		if len(f.EvidenceIDs) == 0 {
			t.Fatalf("finding %q missing evidence", f.Type)
		}
	}
}

func TestGenerate_emptyIsOk(t *testing.T) {
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	summary, findings := review.Generate(review.Input{
		PeriodFrom:              from,
		PeriodTo:                to,
		SustainableFreeCashflow: decimal.RequireFromString("500.00"),
	})
	if len(findings) != 0 {
		t.Fatalf("len = %d", len(findings))
	}
	if summary == "" {
		t.Fatal("empty summary")
	}
}

func TestLargeExpenseThreshold(t *testing.T) {
	got := review.LargeExpenseThreshold(decimal.RequireFromString("200.00"))
	if !got.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("got %s, want 100 floor", got)
	}
	got = review.LargeExpenseThreshold(decimal.RequireFromString("1000.00"))
	if !got.Equal(decimal.RequireFromString("350.00")) {
		t.Fatalf("got %s, want 350", got)
	}
}
