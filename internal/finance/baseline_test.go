package finance_test

import (
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/finance"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestCompute_happyPathCentExact(t *testing.T) {
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

	salaryID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	rentID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	insID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	got := finance.Compute(finance.Input{
		PeriodFrom: from,
		PeriodTo:   to,
		Series: []domain.RecurringSeries{
			{
				ID: salaryID, Kind: domain.RecurringKindIncome,
				Interval: domain.RecurringIntervalMonthly, Status: domain.RecurringStatusActive,
				AmountTypical: decimal.RequireFromString("3500.00"),
			},
			{
				ID: rentID, Kind: domain.RecurringKindFixed,
				Interval: domain.RecurringIntervalMonthly, Status: domain.RecurringStatusActive,
				AmountTypical: decimal.RequireFromString("1200.00"),
			},
			{
				ID: insID, Kind: domain.RecurringKindFixed,
				Interval: domain.RecurringIntervalYearly, Status: domain.RecurringStatusActive,
				AmountTypical: decimal.RequireFromString("600.00"), // → 50/mo
			},
		},
		// Period expenses include rent + variable shopping; insurance not in March.
		CashflowExpense: decimal.RequireFromString("1700.00"),
	})

	assertDec(t, "income", got.RegularMonthlyIncome, "3500.00")
	assertDec(t, "fixed", got.MonthlyFixedCosts, "1200.00")
	assertDec(t, "irregular", got.MonthlyIrregularCosts, "50.00")
	// covered = (1200+50)*1 = 1250; variable = 1700-1250 = 450
	assertDec(t, "variable", got.AvgVariableSpend, "450.00")
	// free = 3500 - 1200 - 50 - 450 = 1800
	assertDec(t, "free", got.SustainableFreeCashflow, "1800.00")
	if got.Confidence != finance.ConfidenceHigh {
		t.Fatalf("confidence = %q", got.Confidence)
	}
	if got.AlgorithmVersion != finance.AlgorithmVersion {
		t.Fatalf("algorithm = %q", got.AlgorithmVersion)
	}
	if len(got.Metrics) != 5 {
		t.Fatalf("metrics = %d", len(got.Metrics))
	}
}

func TestCompute_emptyRecurringLowConfidence(t *testing.T) {
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

	got := finance.Compute(finance.Input{
		PeriodFrom:      from,
		PeriodTo:        to,
		CashflowExpense: decimal.RequireFromString("900.00"),
	})

	assertDec(t, "income", got.RegularMonthlyIncome, "0.00")
	assertDec(t, "fixed", got.MonthlyFixedCosts, "0.00")
	assertDec(t, "irregular", got.MonthlyIrregularCosts, "0.00")
	assertDec(t, "variable", got.AvgVariableSpend, "900.00")
	assertDec(t, "free", got.SustainableFreeCashflow, "-900.00")
	if got.Confidence != finance.ConfidenceLow {
		t.Fatalf("confidence = %q, want low", got.Confidence)
	}
}

func TestCompute_excludesEndedSeries(t *testing.T) {
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

	got := finance.Compute(finance.Input{
		PeriodFrom: from,
		PeriodTo:   to,
		Series: []domain.RecurringSeries{{
			ID: uuid.New(), Kind: domain.RecurringKindFixed,
			Interval: domain.RecurringIntervalMonthly, Status: domain.RecurringStatusEnded,
			AmountTypical: decimal.RequireFromString("999.00"),
		}},
		CashflowExpense: decimal.Zero,
	})
	assertDec(t, "fixed", got.MonthlyFixedCosts, "0.00")
}

func TestCompute_uncertainLowersConfidence(t *testing.T) {
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

	got := finance.Compute(finance.Input{
		PeriodFrom: from,
		PeriodTo:   to,
		Series: []domain.RecurringSeries{{
			ID: uuid.New(), Kind: domain.RecurringKindIncome,
			Interval: domain.RecurringIntervalMonthly, Status: domain.RecurringStatusUncertain,
			AmountTypical: decimal.RequireFromString("3000.00"),
		}},
		CashflowExpense: decimal.RequireFromString("100.00"),
	})
	if got.Confidence != finance.ConfidenceMedium {
		t.Fatalf("confidence = %q, want medium", got.Confidence)
	}
	assertDec(t, "income", got.RegularMonthlyIncome, "3000.00")
}

func TestCompute_quarterlySpread(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC) // ~3 months

	got := finance.Compute(finance.Input{
		PeriodFrom: from,
		PeriodTo:   to,
		Series: []domain.RecurringSeries{{
			ID: uuid.New(), Kind: domain.RecurringKindFixed,
			Interval: domain.RecurringIntervalQuarterly, Status: domain.RecurringStatusActive,
			AmountTypical: decimal.RequireFromString("300.00"), // → 100/mo
		}},
		CashflowExpense: decimal.RequireFromString("300.00"),
	})
	assertDec(t, "irregular", got.MonthlyIrregularCosts, "100.00")
	// covered = 100 * month_span; month_span for Jan1–Mar31
	// days = 90; 90/30.44 ≈ 2.96
	assertDec(t, "variable_nonneg", got.AvgVariableSpend.Abs().Sub(got.AvgVariableSpend), "0.00")
	if got.AvgVariableSpend.IsNegative() {
		t.Fatal("variable should not be negative")
	}
}

func TestCompute_variableFloorWhenRecurringExceedsExpenses(t *testing.T) {
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

	got := finance.Compute(finance.Input{
		PeriodFrom: from,
		PeriodTo:   to,
		Series: []domain.RecurringSeries{{
			ID: uuid.New(), Kind: domain.RecurringKindFixed,
			Interval: domain.RecurringIntervalMonthly, Status: domain.RecurringStatusActive,
			AmountTypical: decimal.RequireFromString("2000.00"),
		}},
		CashflowExpense: decimal.RequireFromString("500.00"),
	})
	assertDec(t, "variable", got.AvgVariableSpend, "0.00")
}

func TestCompute_deterministicEvidenceOrder(t *testing.T) {
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	a := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	b := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")

	in := finance.Input{
		PeriodFrom: from,
		PeriodTo:   to,
		Series: []domain.RecurringSeries{
			{ID: b, Kind: domain.RecurringKindFixed, Interval: domain.RecurringIntervalMonthly,
				Status: domain.RecurringStatusActive, AmountTypical: decimal.RequireFromString("10.00")},
			{ID: a, Kind: domain.RecurringKindFixed, Interval: domain.RecurringIntervalMonthly,
				Status: domain.RecurringStatusActive, AmountTypical: decimal.RequireFromString("20.00")},
		},
		CashflowExpense: decimal.RequireFromString("50.00"),
	}
	first := finance.Compute(in)
	second := finance.Compute(in)
	if first.Metrics[1].EvidenceIDs[0] != a.String() {
		t.Fatalf("evidence order = %v, want %s first", first.Metrics[1].EvidenceIDs, a)
	}
	if first.SustainableFreeCashflow.StringFixed(2) != second.SustainableFreeCashflow.StringFixed(2) {
		t.Fatal("not deterministic")
	}
}

func TestDefaultPeriod_lastCompleteMonth(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	latest := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	from, to, assumption := finance.DefaultPeriod(latest, now)
	if !from.Equal(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("from = %v", from)
	}
	if !to.Equal(time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("to = %v", to)
	}
	if assumption != "period=last_complete_calendar_month" {
		t.Fatalf("assumption = %q", assumption)
	}
}

func TestDefaultPeriod_fallsBackTo90Days(t *testing.T) {
	now := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	latest := time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)
	from, to, assumption := finance.DefaultPeriod(latest, now)
	if !to.Equal(latest) {
		t.Fatalf("to = %v", to)
	}
	if !from.Equal(latest.AddDate(0, 0, -89)) {
		t.Fatalf("from = %v", from)
	}
	if assumption != "period=last_90_days" {
		t.Fatalf("assumption = %q", assumption)
	}
}

func assertDec(t *testing.T, name string, got decimal.Decimal, want string) {
	t.Helper()
	if !got.Equal(decimal.RequireFromString(want)) {
		t.Fatalf("%s = %s, want %s", name, got.StringFixed(2), want)
	}
}
