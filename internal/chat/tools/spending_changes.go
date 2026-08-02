package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/abteilung6/assetagent/internal/llm"
	"github.com/shopspring/decimal"
)

const spendingChangesToolName = "get_spending_changes"

type spendingChangesResult struct {
	OK                bool                   `json:"ok"`
	Currency          string                 `json:"currency"`
	Period            period                 `json:"period"`
	ComparePeriod     period                 `json:"compare_period"`
	AccountsIncluded  []string               `json:"accounts_included"`
	TransfersExcluded bool                   `json:"transfers_excluded"`
	Calculation       string                 `json:"calculation"`
	Confidence        string                 `json:"confidence"`
	DataFreshness     string                 `json:"data_freshness"`
	Assumptions       []string               `json:"assumptions"`
	EvidenceIDs       []string               `json:"evidence_ids"`
	CurrentExpenses   string                 `json:"current_expenses"`
	PreviousExpenses  string                 `json:"previous_expenses"`
	ExpensesDelta     string                 `json:"expenses_delta"`
	Movers            []spendingMoverResult  `json:"movers"`
}

type spendingMoverResult struct {
	Counterparty string `json:"counterparty"`
	Previous     string `json:"previous"`
	Current      string `json:"current"`
	Delta        string `json:"delta"`
}

func spendingChangesTool(reports Reports) toolEntry {
	return toolEntry{
		definition: llm.Tool{
			Name: spendingChangesToolName,
			Description: "Compare household spending between two equal-length periods (current from/to vs the immediately preceding window of the same length). Uses transfer-aware expense totals and top counterparty movers.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"from": {"type": "string", "description": "Start of the current period (YYYY-MM-DD, inclusive)"},
					"to": {"type": "string", "description": "End of the current period (YYYY-MM-DD, inclusive)"}
				},
				"required": ["from", "to"]
			}`),
		},
		handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			return runSpendingChanges(ctx, reports, raw)
		},
	}
}

func runSpendingChanges(ctx context.Context, reports Reports, raw json.RawMessage) (spendingChangesResult, error) {
	if reports == nil {
		return spendingChangesResult{}, errors.New("reports repository is not configured")
	}

	var args cashflowArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return spendingChangesResult{}, fmt.Errorf("invalid arguments: %w", err)
	}
	from, to, err := parseDateRange(args.From, args.To)
	if err != nil {
		return spendingChangesResult{}, err
	}

	days := int(to.Sub(from).Hours()/24) + 1
	prevTo := from.AddDate(0, 0, -1)
	prevFrom := prevTo.AddDate(0, 0, -(days - 1))

	current, err := reports.GetCashflowV2Evidence(ctx, from, to)
	if err != nil {
		return spendingChangesResult{}, err
	}
	previous, err := reports.GetCashflowV2Evidence(ctx, prevFrom, prevTo)
	if err != nil {
		return spendingChangesResult{}, err
	}

	curSpend, err := reports.GetTopCounterparties(ctx, from, to, 10)
	if err != nil {
		return spendingChangesResult{}, err
	}
	prevSpend, err := reports.GetTopCounterparties(ctx, prevFrom, prevTo, 10)
	if err != nil {
		return spendingChangesResult{}, err
	}

	prevByName := map[string]decimal.Decimal{}
	for _, row := range prevSpend {
		prevByName[row.Counterparty] = row.TotalSpent
	}
	movers := make([]spendingMoverResult, 0, len(curSpend))
	for _, row := range curSpend {
		prevAmt := prevByName[row.Counterparty]
		delta := row.TotalSpent.Sub(prevAmt)
		if delta.IsZero() {
			continue
		}
		movers = append(movers, spendingMoverResult{
			Counterparty: row.Counterparty,
			Previous:     decimalString(prevAmt),
			Current:      decimalString(row.TotalSpent),
			Delta:        decimalString(delta),
		})
	}

	evidence := append([]string{}, current.EvidenceIDs...)
	evidence = append(evidence, previous.EvidenceIDs...)

	return spendingChangesResult{
		OK:       true,
		Currency: "EUR",
		Period: period{
			From: args.From,
			To:   args.To,
		},
		ComparePeriod: period{
			From: prevFrom.Format("2006-01-02"),
			To:   prevTo.Format("2006-01-02"),
		},
		AccountsIncluded:  current.AccountsIncluded,
		TransfersExcluded: true,
		Calculation:       "Current-period transfer-aware expenses minus the preceding window of equal length; movers from top counterparties",
		Confidence:        "high",
		DataFreshness:     current.DataFreshness,
		Assumptions: []string{
			"Compare window is the same length immediately before from",
			"Confirmed internal transfers are excluded from expense totals",
		},
		EvidenceIDs:      evidence,
		CurrentExpenses:  decimalString(current.Expenses),
		PreviousExpenses: decimalString(previous.Expenses),
		ExpensesDelta:    decimalString(current.Expenses.Sub(previous.Expenses)),
		Movers:           movers,
	}, nil
}
