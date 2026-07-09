package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/abteilung6/assetagent/internal/llm"
)

const cashflowToolName = "get_monthly_cashflow"

type cashflowArgs struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type cashflowResult struct {
	Income   string `json:"income"`
	Expenses string `json:"expenses"`
	Net      string `json:"net"`
	Currency string `json:"currency"`
}

func cashflowTool(reports Reports) toolEntry {
	return toolEntry{
		definition: llm.Tool{
			Name: cashflowToolName,
			Description: "Get total income, expenses, and net cashflow for a booking-date range.",
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
			return runCashflow(ctx, reports, raw)
		},
	}
}

func runCashflow(ctx context.Context, reports Reports, raw json.RawMessage) (cashflowResult, error) {
	if reports == nil {
		return cashflowResult{}, errors.New("reports repository is not configured")
	}

	var args cashflowArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return cashflowResult{}, fmt.Errorf("invalid arguments: %w", err)
	}

	from, to, err := parseDateRange(args.From, args.To)
	if err != nil {
		return cashflowResult{}, err
	}

	report, err := reports.GetCashflow(ctx, from, to)
	if err != nil {
		return cashflowResult{}, err
	}

	return cashflowResult{
		Income:   decimalString(report.Income),
		Expenses: decimalString(report.Expenses),
		Net:      decimalString(report.Net),
		Currency: "EUR",
	}, nil
}
