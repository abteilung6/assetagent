package classify_test

import (
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/classify"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestDetectRecurringSeries_monthlyRent(t *testing.T) {
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	acc := uuid.New()
	txs := monthlySeries(acc, "Example Landlord", "-1200.00",
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	)

	series := classify.DetectRecurringSeries(txs, nil, now)
	if len(series) != 1 {
		t.Fatalf("len = %d, want 1", len(series))
	}
	s := series[0]
	if s.Interval != domain.RecurringIntervalMonthly {
		t.Fatalf("interval = %q", s.Interval)
	}
	if s.MemberCount != 3 {
		t.Fatalf("members = %d", s.MemberCount)
	}
	if s.Kind != domain.RecurringKindFixed {
		t.Fatalf("kind = %q", s.Kind)
	}
	if s.AmountChanged {
		t.Fatal("expected no amount change")
	}
	if s.Status != domain.RecurringStatusActive {
		t.Fatalf("status = %q", s.Status)
	}
}

func TestDetectRecurringSeries_ignoresOneOff(t *testing.T) {
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	acc := uuid.New()
	txs := []domain.RecurringScanTx{
		{
			ID: uuid.New(), AccountID: acc,
			BookingDate: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
			Amount:      decimal.RequireFromString("-42.00"),
			Counterparty: "One Off Shop",
		},
		{
			ID: uuid.New(), AccountID: acc,
			BookingDate: time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC),
			Amount:      decimal.RequireFromString("-15.00"),
			Counterparty: "Another Place",
		},
	}
	if series := classify.DetectRecurringSeries(txs, nil, now); len(series) != 0 {
		t.Fatalf("len = %d, want 0", len(series))
	}
}

func TestDetectRecurringSeries_flagsAmountStep(t *testing.T) {
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	acc := uuid.New()
	txs := []domain.RecurringScanTx{
		{
			ID: uuid.New(), AccountID: acc,
			BookingDate:  time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC),
			Amount:       decimal.RequireFromString("-89.00"),
			Counterparty: "Example Insurance AG",
		},
		{
			ID: uuid.New(), AccountID: acc,
			BookingDate:  time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC),
			Amount:       decimal.RequireFromString("-89.00"),
			Counterparty: "Example Insurance AG",
		},
		{
			ID: uuid.New(), AccountID: acc,
			BookingDate:  time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC),
			Amount:       decimal.RequireFromString("-99.00"),
			Counterparty: "Example Insurance AG",
		},
	}

	series := classify.DetectRecurringSeries(txs, nil, now)
	if len(series) != 1 {
		t.Fatalf("len = %d, want 1", len(series))
	}
	if !series[0].AmountChanged {
		t.Fatal("expected amount_changed")
	}
	if series[0].Kind != domain.RecurringKindVariableRegular {
		t.Fatalf("kind = %q, want variable_regular", series[0].Kind)
	}
}

func TestDetectRecurringSeries_skipsExistingMembers(t *testing.T) {
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	acc := uuid.New()
	txs := monthlySeries(acc, "Netflix", "-15.99",
		time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
	)
	existing := map[uuid.UUID]struct{}{
		txs[0].ID: {},
		txs[1].ID: {},
		txs[2].ID: {},
	}
	if series := classify.DetectRecurringSeries(txs, existing, now); len(series) != 0 {
		t.Fatalf("len = %d, want 0", len(series))
	}
}

func monthlySeries(acc uuid.UUID, counterparty, amount string, dates ...time.Time) []domain.RecurringScanTx {
	out := make([]domain.RecurringScanTx, len(dates))
	for i, d := range dates {
		out[i] = domain.RecurringScanTx{
			ID:           uuid.New(),
			AccountID:    acc,
			BookingDate:  d,
			Amount:       decimal.RequireFromString(amount),
			Counterparty: counterparty,
		}
	}
	return out
}
