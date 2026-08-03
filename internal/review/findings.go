package review

import (
	"fmt"
	"sort"
	"time"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/shopspring/decimal"
)

const (
	MaxFindings = 3

	FindingFreeCashflowPressure  = "free_cashflow_pressure"
	FindingRecurringAmountChange = "recurring_amount_change"
	FindingLargeExpense          = "large_expense"
	FindingUncertainRecurring    = "uncertain_recurring"
	FindingNeedsReviewResidue    = "needs_review_residue"

	ConfidenceHigh   = "high"
	ConfidenceMedium = "medium"
	ConfidenceLow    = "low"
)

// Finding is one prioritized Money Review insight (domain-authored).
type Finding struct {
	Type               string
	Title              string
	Amount             *decimal.Decimal
	PeriodFrom         time.Time
	PeriodTo           time.Time
	Confidence         string
	EvidenceIDs        []string
	SuggestedActionKey string
	priority           int
}

// Input gathers trusted facts for review generation.
type Input struct {
	PeriodFrom              time.Time
	PeriodTo                time.Time
	SustainableFreeCashflow decimal.Decimal
	Series                  []domain.RecurringSeries
	LargeExpenses           []LargeExpense
	NeedsReviewCount        int
	DataFreshness           string
}

// LargeExpense is a period expense above the anomaly threshold.
type LargeExpense struct {
	TransactionID uuidString
	Label         string
	Amount        decimal.Decimal // absolute positive
}

// uuidString avoids importing uuid in the pure finding core for labels.
type uuidString = string

// Generate builds a deterministic summary and ≤3 findings.
func Generate(in Input) (summary string, findings []Finding) {
	candidates := make([]Finding, 0)

	if in.SustainableFreeCashflow.IsNegative() {
		amount := in.SustainableFreeCashflow
		candidates = append(candidates, Finding{
			Type:               FindingFreeCashflowPressure,
			Title:              fmt.Sprintf("Sustainable free cashflow is negative (%s €/month)", amount.StringFixed(2)),
			Amount:             &amount,
			PeriodFrom:         in.PeriodFrom,
			PeriodTo:           in.PeriodTo,
			Confidence:         ConfidenceHigh,
			EvidenceIDs:        []string{"baseline_free_cashflow"},
			SuggestedActionKey: "reduce_variable_or_fixed",
			priority:           1,
		})
	}

	for _, s := range in.Series {
		if s.Status == domain.RecurringStatusEnded {
			continue
		}
		sid := "series_" + s.ID.String()
		if s.AmountChanged {
			amount := s.AmountLast
			candidates = append(candidates, Finding{
				Type:               FindingRecurringAmountChange,
				Title:              fmt.Sprintf("%s amount changed (typical %s → last %s)", s.DisplayName, s.AmountTypical.StringFixed(2), s.AmountLast.StringFixed(2)),
				Amount:             &amount,
				PeriodFrom:         in.PeriodFrom,
				PeriodTo:           in.PeriodTo,
				Confidence:         ConfidenceMedium,
				EvidenceIDs:        []string{sid},
				SuggestedActionKey: "review_recurring_amount",
				priority:           2,
			})
		}
		if s.Status == domain.RecurringStatusUncertain {
			amount := s.AmountTypical
			candidates = append(candidates, Finding{
				Type:               FindingUncertainRecurring,
				Title:              fmt.Sprintf("%s still needs recurring confirmation", s.DisplayName),
				Amount:             &amount,
				PeriodFrom:         in.PeriodFrom,
				PeriodTo:           in.PeriodTo,
				Confidence:         ConfidenceMedium,
				EvidenceIDs:        []string{sid},
				SuggestedActionKey: "confirm_recurring",
				priority:           4,
			})
		}
	}

	for _, exp := range in.LargeExpenses {
		amount := exp.Amount
		candidates = append(candidates, Finding{
			Type:               FindingLargeExpense,
			Title:              fmt.Sprintf("Large expense: %s (%s)", exp.Label, amount.StringFixed(2)),
			Amount:             &amount,
			PeriodFrom:         in.PeriodFrom,
			PeriodTo:           in.PeriodTo,
			Confidence:         ConfidenceMedium,
			EvidenceIDs:        []string{"tx_" + exp.TransactionID},
			SuggestedActionKey: "explain_one_off",
			priority:           3,
		})
	}

	if in.NeedsReviewCount > 0 {
		candidates = append(candidates, Finding{
			Type:               FindingNeedsReviewResidue,
			Title:              fmt.Sprintf("%d items still need review (transfers, categories, or recurring)", in.NeedsReviewCount),
			PeriodFrom:         in.PeriodFrom,
			PeriodTo:           in.PeriodTo,
			Confidence:         ConfidenceLow,
			EvidenceIDs:        []string{"needs_review_queue"},
			SuggestedActionKey: "clear_needs_review",
			priority:           5,
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].priority != candidates[j].priority {
			return candidates[i].priority < candidates[j].priority
		}
		return candidates[i].Title < candidates[j].Title
	})

	if len(candidates) > MaxFindings {
		candidates = candidates[:MaxFindings]
	}

	summary = buildSummary(in, candidates)
	return summary, candidates
}

// LargeExpenseThreshold matches chat get_anomalies: max(100, 35% of period expenses).
func LargeExpenseThreshold(periodExpenses decimal.Decimal) decimal.Decimal {
	threshold := periodExpenses.Abs().Mul(decimal.NewFromFloat(0.35))
	min := decimal.NewFromInt(100)
	if threshold.LessThan(min) {
		return min
	}
	return threshold
}

func buildSummary(in Input, findings []Finding) string {
	period := fmt.Sprintf("%s – %s",
		in.PeriodFrom.Format("2006-01-02"),
		in.PeriodTo.Format("2006-01-02"),
	)
	if len(findings) == 0 {
		return fmt.Sprintf(
			"Money review for %s: no high-priority findings. Free cashflow %s €/month.",
			period,
			in.SustainableFreeCashflow.StringFixed(2),
		)
	}
	return fmt.Sprintf(
		"Money review for %s: %d finding(s). Free cashflow %s €/month.",
		period,
		len(findings),
		in.SustainableFreeCashflow.StringFixed(2),
	)
}
