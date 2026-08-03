package forecast_test

import (
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/forecast"
	"github.com/shopspring/decimal"
)

func TestProject_deterministicAndMinBalance(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	next := time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)
	in := forecast.Input{
		StartingBalance: decimal.RequireFromString("1000.00"),
		StartDate:       start,
		HorizonDays:     30,
		VariableMonthly: decimal.RequireFromString("304.40"), // → 10/day
		Assumptions:     forecast.DefaultAssumptions(),
		Series: []forecast.SeriesInput{{
			ID: "rent", DisplayName: "Rent", Interval: domain.RecurringIntervalMonthly,
			Kind: domain.RecurringKindFixed, Status: domain.RecurringStatusActive,
			AmountTypical: decimal.RequireFromString("500.00"), NextExpected: &next,
		}},
	}

	a := forecast.Project(in)
	b := forecast.Project(in)
	if a.EndingBalance.StringFixed(2) != b.EndingBalance.StringFixed(2) {
		t.Fatal("not deterministic")
	}
	if a.MinBalance.GreaterThan(a.StartingBalance) {
		t.Fatalf("min %s > start", a.MinBalance)
	}
	// After 30 days variable ≈ 300, plus rent 500 on day 5 → ending around 1000-300-500=200
	if !a.EndingBalance.Equal(decimal.RequireFromString("200.00")) {
		t.Fatalf("ending = %s, want 200.00", a.EndingBalance)
	}
}

func TestProject_disablingSeriesChangesResult(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	next := time.Date(2026, 4, 5, 0, 0, 0, 0, time.UTC)
	base := forecast.Input{
		StartingBalance: decimal.RequireFromString("1000.00"),
		StartDate:       start,
		HorizonDays:     30,
		VariableMonthly: decimal.Zero,
		Assumptions:     forecast.DefaultAssumptions(),
		Series: []forecast.SeriesInput{{
			ID: "rent", DisplayName: "Rent", Interval: domain.RecurringIntervalMonthly,
			Kind: domain.RecurringKindFixed, Status: domain.RecurringStatusActive,
			AmountTypical: decimal.RequireFromString("500.00"), NextExpected: &next,
		}},
	}
	with := forecast.Project(base)
	base.Assumptions.DisabledSeriesIDs = []string{"rent"}
	without := forecast.Project(base)
	if with.EndingBalance.Equal(without.EndingBalance) {
		t.Fatal("expected disabling rent to change ending balance")
	}
	if !without.EndingBalance.Equal(decimal.RequireFromString("1000.00")) {
		t.Fatalf("without rent ending = %s", without.EndingBalance)
	}
}

func TestApplyScenario_newMonthlyObligation(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	base := forecast.Input{
		StartingBalance: decimal.RequireFromString("2000.00"),
		StartDate:       start,
		HorizonDays:     90,
		VariableMonthly: decimal.Zero,
		Assumptions:     forecast.DefaultAssumptions(),
	}
	amt := decimal.RequireFromString("100.00")
	startDate := start
	got, err := forecast.ApplyScenario(base, forecast.ScenarioNewMonthlyObligation, forecast.ScenarioParams{
		MonthlyAmount: &amt,
		StartDate:     &startDate,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.FreeCashflowDelta.Equal(decimal.RequireFromString("-100.00")) {
		t.Fatalf("delta = %s", got.FreeCashflowDelta)
	}
	if got.MinBalance.GreaterThanOrEqual(got.BaselineMinBalance) && got.MinBalance.Equal(got.BaselineMinBalance) {
		// obligation should reduce min vs baseline of flat 2000
	}
	if got.MinBalance.Equal(decimal.RequireFromString("2000.00")) {
		t.Fatal("expected obligation to reduce min balance")
	}
}

func TestApplyScenario_oneOffPlusGoal(t *testing.T) {
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	by := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	base := forecast.Input{
		StartingBalance: decimal.RequireFromString("5000.00"),
		StartDate:       start,
		HorizonDays:     90,
		VariableMonthly: decimal.Zero,
		Assumptions:     forecast.DefaultAssumptions(),
	}
	oneOff := decimal.RequireFromString("1000.00")
	goal := decimal.RequireFromString("3000.00")
	got, err := forecast.ApplyScenario(base, forecast.ScenarioOneOffPlusGoal, forecast.ScenarioParams{
		OneOffAmount: &oneOff,
		GoalAmount:   &goal,
		ByDate:       &by,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.GoalFeasible == nil || !*got.GoalFeasible {
		t.Fatal("expected goal feasible")
	}
}

func TestApplyScenario_rejectsUnknown(t *testing.T) {
	_, err := forecast.ApplyScenario(forecast.Input{Assumptions: forecast.DefaultAssumptions()}, "nope", forecast.ScenarioParams{})
	if err == nil {
		t.Fatal("expected error")
	}
}
