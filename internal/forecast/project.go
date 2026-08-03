package forecast

import (
	"fmt"
	"sort"
	"time"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/shopspring/decimal"
)

const (
	DefaultHorizonDays = 90
	AlgorithmVersion   = "forecast.v1"
)

// Assumptions controls which cashflows enter the projection.
type Assumptions struct {
	DisabledSeriesIDs []string `json:"disabled_series_ids"`
	IncludeVariable   bool     `json:"include_variable"`
	IncludeUncertain  bool     `json:"include_uncertain"`
}

// DefaultAssumptions enables variable spend and uncertain series.
func DefaultAssumptions() Assumptions {
	return Assumptions{
		DisabledSeriesIDs: []string{},
		IncludeVariable:   true,
		IncludeUncertain:  true,
	}
}

// Point is one projected balance sample.
type Point struct {
	Date    time.Time       `json:"date"`
	Balance decimal.Decimal `json:"balance"`
}

// SeriesInput is the recurring slice needed for projection.
type SeriesInput struct {
	ID            string
	DisplayName   string
	Interval      string
	Kind          string
	Status        string
	AmountTypical decimal.Decimal
	NextExpected  *time.Time
}

// Input drives a deterministic projection.
type Input struct {
	StartingBalance decimal.Decimal
	StartDate       time.Time
	HorizonDays     int
	Series          []SeriesInput
	VariableMonthly decimal.Decimal
	Assumptions     Assumptions
}

// Result is the full projection artifact (before persistence).
type Result struct {
	HorizonDays      int
	StartingBalance  decimal.Decimal
	MinBalance       decimal.Decimal
	EndingBalance    decimal.Decimal
	Points           []Point
	Assumptions      Assumptions
	AlgorithmVersion string
}

// Project computes a deterministic balance path.
func Project(in Input) Result {
	horizon := in.HorizonDays
	if horizon <= 0 {
		horizon = DefaultHorizonDays
	}
	start := dateOnly(in.StartDate)
	assumptions := in.Assumptions
	if assumptions.DisabledSeriesIDs == nil {
		assumptions.DisabledSeriesIDs = []string{}
	}

	disabled := make(map[string]struct{}, len(assumptions.DisabledSeriesIDs))
	for _, id := range assumptions.DisabledSeriesIDs {
		disabled[id] = struct{}{}
	}

	dailyVariable := decimal.Zero
	if assumptions.IncludeVariable && !in.VariableMonthly.IsZero() {
		dailyVariable = in.VariableMonthly.Abs().Div(decimal.RequireFromString("30.44")).Round(6)
	}

	events := scheduledEvents(in.Series, start, horizon, assumptions.IncludeUncertain, disabled)

	balance := in.StartingBalance.Round(2)
	minBal := balance
	points := make([]Point, 0, horizon/7+2)
	points = append(points, Point{Date: start, Balance: balance})

	for day := 1; day <= horizon; day++ {
		d := start.AddDate(0, 0, day)
		balance = balance.Sub(dailyVariable)
		if deltas, ok := events[d]; ok {
			for _, delta := range deltas {
				balance = balance.Add(delta)
			}
		}
		balance = balance.Round(2)
		if balance.LessThan(minBal) {
			minBal = balance
		}
		if day == horizon || day%7 == 0 {
			points = append(points, Point{Date: d, Balance: balance})
		}
	}

	return Result{
		HorizonDays:      horizon,
		StartingBalance:  in.StartingBalance.Round(2),
		MinBalance:       minBal.Round(2),
		EndingBalance:    balance.Round(2),
		Points:           points,
		Assumptions:      assumptions,
		AlgorithmVersion: AlgorithmVersion,
	}
}

func scheduledEvents(
	series []SeriesInput,
	start time.Time,
	horizon int,
	includeUncertain bool,
	disabled map[string]struct{},
) map[time.Time][]decimal.Decimal {
	end := start.AddDate(0, 0, horizon)
	out := make(map[time.Time][]decimal.Decimal)

	for _, s := range series {
		if _, skip := disabled[s.ID]; skip {
			continue
		}
		if s.Status == domain.RecurringStatusEnded {
			continue
		}
		if s.Status == domain.RecurringStatusUncertain && !includeUncertain {
			continue
		}
		amount := s.AmountTypical.Abs()
		if amount.IsZero() {
			continue
		}
		if s.Kind != domain.RecurringKindIncome {
			amount = amount.Neg()
		}

		first := start
		if s.NextExpected != nil {
			first = dateOnly(*s.NextExpected)
			if first.Before(start) {
				first = advanceToOnOrAfter(first, s.Interval, start)
			}
		} else {
			first = start.AddDate(0, 0, cadenceDays(s.Interval))
		}

		for d := first; !d.After(end); d = advance(d, s.Interval) {
			if d.Before(start) {
				continue
			}
			out[d] = append(out[d], amount)
		}
	}
	return out
}

func cadenceDays(interval string) int {
	switch interval {
	case domain.RecurringIntervalQuarterly:
		return 91
	case domain.RecurringIntervalYearly:
		return 365
	default:
		return 30
	}
}

func advance(d time.Time, interval string) time.Time {
	switch interval {
	case domain.RecurringIntervalQuarterly:
		return d.AddDate(0, 3, 0)
	case domain.RecurringIntervalYearly:
		return d.AddDate(1, 0, 0)
	default:
		return d.AddDate(0, 1, 0)
	}
}

func advanceToOnOrAfter(from time.Time, interval string, target time.Time) time.Time {
	d := from
	for d.Before(target) {
		d = advance(d, interval)
	}
	return d
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// SeriesFromDomain maps domain recurring rows into projection inputs.
func SeriesFromDomain(series []domain.RecurringSeries) []SeriesInput {
	out := make([]SeriesInput, 0, len(series))
	for _, s := range series {
		out = append(out, SeriesInput{
			ID:            s.ID.String(),
			DisplayName:   s.DisplayName,
			Interval:      s.Interval,
			Kind:          s.Kind,
			Status:        s.Status,
			AmountTypical: s.AmountTypical,
			NextExpected:  s.NextExpected,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ScenarioKind is one of the three MVP scenario types.
type ScenarioKind string

const (
	ScenarioNewMonthlyObligation ScenarioKind = "new_monthly_obligation"
	ScenarioIncomeGap            ScenarioKind = "income_gap"
	ScenarioOneOffPlusGoal       ScenarioKind = "one_off_plus_goal"
)

// ScenarioParams are typed inputs (validated by kind).
type ScenarioParams struct {
	MonthlyAmount      *decimal.Decimal `json:"monthly_amount,omitempty"`
	StartDate          *time.Time       `json:"start_date,omitempty"`
	MonthlyIncomeDelta *decimal.Decimal `json:"monthly_income_delta,omitempty"`
	Months             *int             `json:"months,omitempty"`
	OneOffAmount       *decimal.Decimal `json:"one_off_amount,omitempty"`
	GoalAmount         *decimal.Decimal `json:"goal_amount,omitempty"`
	ByDate             *time.Time       `json:"by_date,omitempty"`
}

// ScenarioResult is the computed outcome for a typed scenario.
type ScenarioResult struct {
	Kind               ScenarioKind     `json:"kind"`
	MinBalance         decimal.Decimal  `json:"min_balance"`
	EndingBalance      decimal.Decimal  `json:"ending_balance"`
	FreeCashflowDelta  decimal.Decimal  `json:"free_cashflow_delta"`
	GoalFeasible       *bool            `json:"goal_feasible,omitempty"`
	ProjectedAtByDate  *decimal.Decimal `json:"projected_at_by_date,omitempty"`
	BaselineMinBalance decimal.Decimal  `json:"baseline_min_balance"`
	Notes              []string         `json:"notes"`
}

// ApplyScenario overlays a typed scenario onto a base forecast input and recomputes.
func ApplyScenario(base Input, kind ScenarioKind, params ScenarioParams) (ScenarioResult, error) {
	baseResult := Project(base)
	modified := base
	modified.Series = append([]SeriesInput{}, base.Series...)
	notes := []string{}
	freeDelta := decimal.Zero

	switch kind {
	case ScenarioNewMonthlyObligation:
		if params.MonthlyAmount == nil || params.StartDate == nil {
			return ScenarioResult{}, fmt.Errorf("new_monthly_obligation requires monthly_amount and start_date")
		}
		amt := params.MonthlyAmount.Abs()
		freeDelta = amt.Neg()
		start := dateOnly(*params.StartDate)
		modified.Series = append(modified.Series, SeriesInput{
			ID:            "scenario_new_obligation",
			DisplayName:   "Scenario obligation",
			Interval:      domain.RecurringIntervalMonthly,
			Kind:          domain.RecurringKindFixed,
			Status:        domain.RecurringStatusActive,
			AmountTypical: amt,
			NextExpected:  &start,
		})
		notes = append(notes, fmt.Sprintf("Added monthly obligation of %s from %s", amt.StringFixed(2), start.Format("2006-01-02")))

	case ScenarioIncomeGap:
		if params.MonthlyIncomeDelta == nil || params.Months == nil || *params.Months <= 0 {
			return ScenarioResult{}, fmt.Errorf("income_gap requires monthly_income_delta and months > 0")
		}
		delta := *params.MonthlyIncomeDelta
		freeDelta = delta
		months := *params.Months
		start := dateOnly(base.StartDate)
		kindFor := domain.RecurringKindIncome
		amt := delta.Abs()
		if delta.IsNegative() {
			kindFor = domain.RecurringKindFixed
		}
		for i := 0; i < months; i++ {
			d := start.AddDate(0, i, 0)
			modified.Series = append(modified.Series, SeriesInput{
				ID:            fmt.Sprintf("scenario_income_gap_%d", i),
				DisplayName:   "Income gap",
				Interval:      domain.RecurringIntervalYearly, // once per NextExpected
				Kind:          kindFor,
				Status:        domain.RecurringStatusActive,
				AmountTypical: amt,
				NextExpected:  &d,
			})
		}
		notes = append(notes, fmt.Sprintf("Income change %s/month for %d months", delta.StringFixed(2), months))

	case ScenarioOneOffPlusGoal:
		if params.OneOffAmount == nil || params.GoalAmount == nil || params.ByDate == nil {
			return ScenarioResult{}, fmt.Errorf("one_off_plus_goal requires one_off_amount, goal_amount, and by_date")
		}
		oneOff := params.OneOffAmount.Abs()
		goal := params.GoalAmount.Abs()
		by := dateOnly(*params.ByDate)
		modified.StartingBalance = base.StartingBalance.Sub(oneOff)
		notes = append(notes, fmt.Sprintf("One-off cost %s; goal reserve %s by %s", oneOff.StringFixed(2), goal.StringFixed(2), by.Format("2006-01-02")))

		result := Project(modified)
		atBy := balanceOnOrAfter(modified, by)
		feasible := atBy.GreaterThanOrEqual(goal)
		return ScenarioResult{
			Kind:               kind,
			MinBalance:         result.MinBalance,
			EndingBalance:      result.EndingBalance,
			FreeCashflowDelta:  decimal.Zero,
			GoalFeasible:       &feasible,
			ProjectedAtByDate:  &atBy,
			BaselineMinBalance: baseResult.MinBalance,
			Notes:              notes,
		}, nil

	default:
		return ScenarioResult{}, fmt.Errorf("unknown scenario kind %q", kind)
	}

	result := Project(modified)
	return ScenarioResult{
		Kind:               kind,
		MinBalance:         result.MinBalance,
		EndingBalance:      result.EndingBalance,
		FreeCashflowDelta:  freeDelta.Round(2),
		BaselineMinBalance: baseResult.MinBalance,
		Notes:              notes,
	}, nil
}

func balanceOnOrAfter(in Input, target time.Time) decimal.Decimal {
	horizon := in.HorizonDays
	if horizon <= 0 {
		horizon = DefaultHorizonDays
	}
	start := dateOnly(in.StartDate)
	days := int(target.Sub(start).Hours()/24)
	if days < 0 {
		days = 0
	}
	if days > horizon {
		days = horizon
	}
	tmp := in
	tmp.HorizonDays = days
	return Project(tmp).EndingBalance
}
