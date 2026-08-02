package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/llm"
	"github.com/shopspring/decimal"
)

const anomaliesToolName = "get_anomalies"

type anomaliesResult struct {
	OK                bool             `json:"ok"`
	Currency          string           `json:"currency"`
	Period            period           `json:"period"`
	AccountsIncluded  []string         `json:"accounts_included"`
	TransfersExcluded bool             `json:"transfers_excluded"`
	Calculation       string           `json:"calculation"`
	Confidence        string           `json:"confidence"`
	DataFreshness     string           `json:"data_freshness"`
	Assumptions       []string         `json:"assumptions"`
	EvidenceIDs       []string         `json:"evidence_ids"`
	Count             int              `json:"count"`
	Findings          []anomalyFinding `json:"findings"`
}

type anomalyFinding struct {
	Type        string   `json:"type"`
	Title       string   `json:"title"`
	Amount      string   `json:"amount,omitempty"`
	EvidenceIDs []string `json:"evidence_ids"`
}

func anomaliesTool(source recurringSeriesSource, reports Reports, lister TransactionLister) toolEntry {
	return toolEntry{
		definition: llm.Tool{
			Name: anomaliesToolName,
			Description: "Find notable money anomalies in a period: recurring amount steps, uncertain recurring series, and large one-off expenses relative to other spending.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"from": {"type": "string", "description": "Start date (YYYY-MM-DD, inclusive)"},
					"to": {"type": "string", "description": "End date (YYYY-MM-DD, inclusive)"}
				},
				"required": ["from", "to"]
			}`),
		},
		handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			return runAnomalies(ctx, source, reports, lister, raw)
		},
	}
}

func runAnomalies(
	ctx context.Context,
	source recurringSeriesSource,
	reports Reports,
	lister TransactionLister,
	raw json.RawMessage,
) (anomaliesResult, error) {
	if reports == nil {
		return anomaliesResult{}, errors.New("reports repository is not configured")
	}

	var args cashflowArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return anomaliesResult{}, fmt.Errorf("invalid arguments: %w", err)
	}
	from, to, err := parseDateRange(args.From, args.To)
	if err != nil {
		return anomaliesResult{}, err
	}

	evidence, err := reports.GetCashflowV2Evidence(ctx, from, to)
	if err != nil {
		return anomaliesResult{}, err
	}

	findings := make([]anomalyFinding, 0)
	ids := make([]string, 0)

	if source != nil {
		if _, err := source.Scan(ctx); err == nil {
			series, err := source.List(ctx)
			if err == nil {
				for _, s := range series {
					if s.Status == domain.RecurringStatusEnded {
						continue
					}
					sid := "series_" + s.ID.String()
					if s.AmountChanged {
						findings = append(findings, anomalyFinding{
							Type:        "recurring_amount_change",
							Title:       fmt.Sprintf("%s amount changed (typical %s → last %s)", s.DisplayName, s.AmountTypical.StringFixed(2), s.AmountLast.StringFixed(2)),
							Amount:      decimalString(s.AmountLast),
							EvidenceIDs: []string{sid},
						})
						ids = append(ids, sid)
					}
					if s.Status == domain.RecurringStatusUncertain {
						findings = append(findings, anomalyFinding{
							Type:        "uncertain_recurring",
							Title:       fmt.Sprintf("%s recurring series still needs review", s.DisplayName),
							Amount:      decimalString(s.AmountTypical),
							EvidenceIDs: []string{sid},
						})
						ids = append(ids, sid)
					}
				}
			}
		}
	}

	if lister != nil && !evidence.Expenses.IsZero() {
		listed, err := lister.List(ctx, domain.ListParams{
			Limit:    20,
			FromDate: &from,
			ToDate:   &to,
			Sort:     domain.SortAmount,
			SortAsc:  true, // most negative first
		})
		if err == nil {
			threshold := evidence.Expenses.Mul(decimal.NewFromFloat(0.35))
			if threshold.LessThan(decimal.NewFromInt(100)) {
				threshold = decimal.NewFromInt(100)
			}
			for _, tx := range listed.Transactions {
				if !tx.Amount.IsNegative() {
					continue
				}
				abs := tx.Amount.Abs()
				if abs.LessThan(threshold) {
					continue
				}
				eid := "tx_" + tx.ID.String()
				findings = append(findings, anomalyFinding{
					Type:        "large_expense",
					Title:       fmt.Sprintf("Large expense %s (%s)", labelTx(tx), abs.StringFixed(2)),
					Amount:      decimalString(abs),
					EvidenceIDs: []string{eid},
				})
				ids = append(ids, eid)
			}
		}
	}

	sort.SliceStable(findings, func(i, j int) bool {
		return findings[i].Type < findings[j].Type
	})

	confidence := "medium"
	if len(findings) == 0 {
		confidence = "high"
	}

	return anomaliesResult{
		OK:                true,
		Currency:          "EUR",
		Period:            period{From: args.From, To: args.To},
		AccountsIncluded:  evidence.AccountsIncluded,
		TransfersExcluded: true,
		Calculation:       "Recurring amount steps, uncertain series, and expenses ≥ max(100, 35% of period expenses)",
		Confidence:        confidence,
		DataFreshness:     evidence.DataFreshness,
		Assumptions: []string{
			"Confirmed internal transfers are excluded from the large-expense pool where cashflow v2 evidence is used",
			"Thresholds are heuristic, not user-configured",
		},
		EvidenceIDs: ids,
		Count:       len(findings),
		Findings:    findings,
	}, nil
}

func labelTx(tx domain.Transaction) string {
	if tx.Counterparty != "" {
		return tx.Counterparty
	}
	if tx.Purpose != "" {
		return tx.Purpose
	}
	return tx.BookingText
}
