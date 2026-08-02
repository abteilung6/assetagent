package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/llm"
	"github.com/shopspring/decimal"
)

const recurringCostsToolName = "get_recurring_costs"

type recurringSeriesSource interface {
	Scan(ctx context.Context) (domain.RecurringScanResult, error)
	List(ctx context.Context) ([]domain.RecurringSeries, error)
}

type recurringCostsResult struct {
	OK                bool                    `json:"ok"`
	Currency          string                  `json:"currency"`
	Period            period                  `json:"period"`
	AccountsIncluded  []string                `json:"accounts_included"`
	TransfersExcluded bool                    `json:"transfers_excluded"`
	Calculation       string                  `json:"calculation"`
	Confidence        string                  `json:"confidence"`
	DataFreshness     string                  `json:"data_freshness"`
	Assumptions       []string                `json:"assumptions"`
	EvidenceIDs       []string                `json:"evidence_ids"`
	Count             int                     `json:"count"`
	MonthlyTotal      string                  `json:"monthly_total"`
	Series            []recurringSeriesResult `json:"series"`
}

type recurringSeriesResult struct {
	DisplayName   string `json:"display_name"`
	Interval      string `json:"interval"`
	Kind          string `json:"kind"`
	Status        string `json:"status"`
	AmountTypical string `json:"amount_typical"`
	AmountLast    string `json:"amount_last"`
	AmountChanged bool   `json:"amount_changed"`
	MemberCount   int    `json:"member_count"`
	MonthlyAmount string `json:"monthly_amount"`
}

func recurringCostsTool(source recurringSeriesSource, reports Reports) toolEntry {
	return toolEntry{
		definition: llm.Tool{
			Name: recurringCostsToolName,
			Description: "List detected recurring payments (rent, subscriptions, insurance) with typical amounts. Prefer this for questions about regular monthly costs.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"from": {"type": "string", "description": "Optional start date (YYYY-MM-DD) used only for evidence freshness context"},
					"to": {"type": "string", "description": "Optional end date (YYYY-MM-DD) used only for evidence freshness context"}
				}
			}`),
		},
		handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			return runRecurringCosts(ctx, source, reports, raw)
		},
	}
}

func runRecurringCosts(
	ctx context.Context,
	source recurringSeriesSource,
	reports Reports,
	raw json.RawMessage,
) (recurringCostsResult, error) {
	if source == nil {
		return recurringCostsResult{}, errors.New("recurring service is not configured")
	}

	var args cashflowArgs
	_ = json.Unmarshal(raw, &args)
	fromRaw, toRaw := args.From, args.To
	if fromRaw == "" {
		fromRaw = "1970-01-01"
	}
	if toRaw == "" {
		toRaw = "2999-12-31"
	}
	from, to, err := parseDateRange(fromRaw, toRaw)
	if err != nil {
		return recurringCostsResult{}, err
	}

	if _, err := source.Scan(ctx); err != nil {
		return recurringCostsResult{}, fmt.Errorf("recurring scan: %w", err)
	}
	all, err := source.List(ctx)
	if err != nil {
		return recurringCostsResult{}, err
	}

	accounts := []string{}
	freshness := ""
	if reports != nil {
		if evidence, err := reports.GetCashflowV2Evidence(ctx, from, to); err == nil {
			accounts = evidence.AccountsIncluded
			freshness = evidence.DataFreshness
		}
	}

	seriesOut := make([]recurringSeriesResult, 0, len(all))
	evidenceIDs := make([]string, 0, len(all))
	monthlyTotal := decimal.Zero
	confidence := "high"

	for _, s := range all {
		if s.Status == domain.RecurringStatusEnded {
			continue
		}
		if s.Kind == domain.RecurringKindIncome {
			continue
		}
		monthly := monthlyEquivalent(s.AmountTypical, s.Interval)
		monthlyTotal = monthlyTotal.Add(monthly)
		if s.Status == domain.RecurringStatusUncertain || s.Uncertainty == domain.RecurringUncertaintyHigh {
			confidence = "medium"
		}
		seriesOut = append(seriesOut, recurringSeriesResult{
			DisplayName:   s.DisplayName,
			Interval:      s.Interval,
			Kind:          s.Kind,
			Status:        s.Status,
			AmountTypical: decimalString(s.AmountTypical),
			AmountLast:    decimalString(s.AmountLast),
			AmountChanged: s.AmountChanged,
			MemberCount:   s.MemberCount,
			MonthlyAmount: decimalString(monthly),
		})
		evidenceIDs = append(evidenceIDs, "series_"+s.ID.String())
	}

	return recurringCostsResult{
		OK:               true,
		Currency:         "EUR",
		Period:           period{From: fromRaw, To: toRaw},
		AccountsIncluded: accounts,
		TransfersExcluded: true,
		Calculation:      "Detected recurring expense series; monthly_total annualizes quarterly/yearly to a monthly equivalent",
		Confidence:       confidence,
		DataFreshness:    freshness,
		Assumptions: []string{
			"Income series are omitted",
			"Ended series are omitted",
			"Uncertain series are included until the user dismisses them",
		},
		EvidenceIDs:  evidenceIDs,
		Count:        len(seriesOut),
		MonthlyTotal: decimalString(monthlyTotal),
		Series:       seriesOut,
	}, nil
}

func monthlyEquivalent(amount decimal.Decimal, interval string) decimal.Decimal {
	switch interval {
	case domain.RecurringIntervalQuarterly:
		return amount.Div(decimal.NewFromInt(3))
	case domain.RecurringIntervalYearly:
		return amount.Div(decimal.NewFromInt(12))
	default:
		return amount
	}
}
