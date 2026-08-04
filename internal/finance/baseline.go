package finance

import (
	"fmt"
	"sort"
	"time"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/shopspring/decimal"
)

const AlgorithmVersion = "baseline.v1"

const (
	MetricRegularMonthlyIncome  = "regular_monthly_income"
	MetricMonthlyFixedCosts     = "monthly_fixed_costs"
	MetricMonthlyIrregularCosts = "monthly_irregular_costs"
	MetricAvgVariableSpend      = "avg_variable_spend"
	MetricSustainableFreeCash   = "sustainable_free_cashflow"

	ConfidenceHigh   = "high"
	ConfidenceMedium = "medium"
	ConfidenceLow    = "low"
)

// Input is everything the pure baseline calculator needs.
type Input struct {
	PeriodFrom      time.Time
	PeriodTo        time.Time
	Series          []domain.RecurringSeries
	CashflowExpense decimal.Decimal // transfer-aware expense total (positive)
	Assumptions     []string
}

// MetricEvidence describes one of the five baseline numbers.
type MetricEvidence struct {
	Key         string
	Value       decimal.Decimal
	Calculation string
	Confidence  string
	EvidenceIDs []string
	Assumptions []string
}

// Baseline is the deterministic five-number household snapshot.
type Baseline struct {
	PeriodFrom              time.Time
	PeriodTo                time.Time
	AlgorithmVersion        string
	RegularMonthlyIncome    decimal.Decimal
	MonthlyFixedCosts       decimal.Decimal
	MonthlyIrregularCosts   decimal.Decimal
	AvgVariableSpend        decimal.Decimal
	SustainableFreeCashflow decimal.Decimal
	Confidence              string
	Assumptions             []string
	Metrics                 []MetricEvidence
	MonthSpan               decimal.Decimal
}

// Compute derives the five baseline metrics. Same inputs always yield the same outputs.
func Compute(in Input) Baseline {
	from := dateOnly(in.PeriodFrom)
	to := dateOnly(in.PeriodTo)
	months := inclusiveMonthSpan(from, to)

	assumptions := append([]string{}, in.Assumptions...)
	assumptions = append(assumptions,
		fmt.Sprintf("algorithm=%s", AlgorithmVersion),
		fmt.Sprintf("month_span=%s", months.StringFixed(2)),
	)

	var (
		income      decimal.Decimal
		fixed       decimal.Decimal
		irregular   decimal.Decimal
		incomeIDs   []string
		fixedIDs    []string
		irregularIDs []string
		uncertain   int
		included    int
	)

	for _, s := range in.Series {
		if s.Status == domain.RecurringStatusEnded {
			continue
		}
		monthly := monthlyEquivalent(s.AmountTypical, s.Interval)
		if monthly.IsZero() {
			continue
		}
		included++
		if s.Status == domain.RecurringStatusUncertain {
			uncertain++
		}
		id := s.ID.String()

		switch s.Kind {
		case domain.RecurringKindIncome:
			income = income.Add(monthly)
			incomeIDs = append(incomeIDs, id)
		default:
			// fixed + variable_regular expenses
			switch s.Interval {
			case domain.RecurringIntervalQuarterly, domain.RecurringIntervalYearly:
				irregular = irregular.Add(monthly)
				irregularIDs = append(irregularIDs, id)
			default:
				fixed = fixed.Add(monthly)
				fixedIDs = append(fixedIDs, id)
			}
		}
	}
	sort.Strings(incomeIDs)
	sort.Strings(fixedIDs)
	sort.Strings(irregularIDs)

	expense := in.CashflowExpense.Abs()
	covered := fixed.Add(irregular).Mul(months)
	variableTotal := expense.Sub(covered)
	if variableTotal.IsNegative() {
		assumptions = append(assumptions, "variable_floor=0 (recurring covered exceeded period expenses)")
		variableTotal = decimal.Zero
	}
	avgVariable := variableTotal.Div(months).Round(2)

	free := income.Sub(fixed).Sub(irregular).Sub(avgVariable).Round(2)
	income = income.Round(2)
	fixed = fixed.Round(2)
	irregular = irregular.Round(2)

	overall := overallConfidence(included, uncertain)
	incomeConf := metricConfidence(len(incomeIDs), uncertainIncome(in.Series), overall)
	fixedConf := metricConfidence(len(fixedIDs), 0, overall)
	irregConf := metricConfidence(len(irregularIDs), 0, overall)
	varConf := ConfidenceMedium
	if included == 0 {
		varConf = ConfidenceLow
	} else if uncertain == 0 && !expense.IsZero() {
		varConf = ConfidenceHigh
	}

	metrics := []MetricEvidence{
		{
			Key:         MetricRegularMonthlyIncome,
			Value:       income,
			Calculation: "Sum of monthly-equivalent active/uncertain income series",
			Confidence:  incomeConf,
			EvidenceIDs: incomeIDs,
		},
		{
			Key:         MetricMonthlyFixedCosts,
			Value:       fixed,
			Calculation: "Sum of monthly-equivalent recurring expense series (monthly cadence)",
			Confidence:  fixedConf,
			EvidenceIDs: fixedIDs,
		},
		{
			Key:         MetricMonthlyIrregularCosts,
			Value:       irregular,
			Calculation: "Quarterly/yearly recurring expenses spread into a monthly amount",
			Confidence:  irregConf,
			EvidenceIDs: irregularIDs,
		},
		{
			Key:   MetricAvgVariableSpend,
			Value: avgVariable,
			Calculation: fmt.Sprintf(
				"Transfer-aware expenses %s minus recurring cover %s, divided by %s months",
				expense.StringFixed(2),
				covered.StringFixed(2),
				months.StringFixed(2),
			),
			Confidence:  varConf,
			EvidenceIDs: append(append([]string{}, fixedIDs...), irregularIDs...),
			Assumptions: []string{"cashflow_v2_expenses_exclude_confirmed_transfers"},
		},
		{
			Key:         MetricSustainableFreeCash,
			Value:       free,
			Calculation: "Income minus fixed costs, irregular costs, and average variable spend",
			Confidence:  overall,
			EvidenceIDs: nil,
		},
	}

	return Baseline{
		PeriodFrom:              from,
		PeriodTo:                to,
		AlgorithmVersion:        AlgorithmVersion,
		RegularMonthlyIncome:    income,
		MonthlyFixedCosts:       fixed,
		MonthlyIrregularCosts:   irregular,
		AvgVariableSpend:        avgVariable,
		SustainableFreeCashflow: free,
		Confidence:              overall,
		Assumptions:             assumptions,
		Metrics:                 metrics,
		MonthSpan:               months,
	}
}

func monthlyEquivalent(amount decimal.Decimal, interval string) decimal.Decimal {
	abs := amount.Abs()
	switch interval {
	case domain.RecurringIntervalQuarterly:
		return abs.Div(decimal.NewFromInt(3)).Round(4)
	case domain.RecurringIntervalYearly:
		return abs.Div(decimal.NewFromInt(12)).Round(4)
	default:
		return abs
	}
}

func inclusiveMonthSpan(from, to time.Time) decimal.Decimal {
	from = dateOnly(from)
	to = dateOnly(to)
	if to.Before(from) {
		return decimal.NewFromInt(1)
	}
	// Full calendar-month windows (1st → last day of same or later month).
	if from.Day() == 1 {
		nextAfterTo := time.Date(to.Year(), to.Month()+1, 1, 0, 0, 0, 0, time.UTC)
		monthEnd := nextAfterTo.AddDate(0, 0, -1)
		if to.Equal(monthEnd) {
			months := (to.Year()-from.Year())*12 + int(to.Month()-from.Month()) + 1
			if months >= 1 {
				return decimal.NewFromInt(int64(months))
			}
		}
	}
	days := int(to.Sub(from).Hours()/24) + 1
	if days < 1 {
		days = 1
	}
	span := decimal.NewFromInt(int64(days)).Div(decimal.RequireFromString("30.44")).Round(2)
	if span.LessThan(decimal.NewFromInt(1)) {
		return decimal.NewFromInt(1)
	}
	return span
}

func overallConfidence(included, uncertain int) string {
	if included == 0 {
		return ConfidenceLow
	}
	if uncertain > 0 {
		return ConfidenceMedium
	}
	return ConfidenceHigh
}

func metricConfidence(count, uncertainInMetric int, overall string) string {
	if count == 0 {
		return ConfidenceLow
	}
	if uncertainInMetric > 0 {
		return ConfidenceMedium
	}
	return overall
}

func uncertainIncome(series []domain.RecurringSeries) int {
	n := 0
	for _, s := range series {
		if s.Kind == domain.RecurringKindIncome && s.Status == domain.RecurringStatusUncertain {
			n++
		}
	}
	return n
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
