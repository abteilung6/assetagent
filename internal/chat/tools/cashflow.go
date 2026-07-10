package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/abteilung6/assetagent/internal/llm"
)

const cashflowToolName = "get_cashflow"

type cashflowArgs struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type cashflowResult struct {
	OK       bool   `json:"ok"`
	From     string `json:"from"`
	To       string `json:"to"`
	Income   string `json:"income"`
	Expenses string `json:"expenses"`
	Net      string `json:"net"`
	Currency string `json:"currency"`
}

func cashflowTool(reports Reports) toolEntry {
	return toolEntry{
		definition: llm.Tool{
			Name: cashflowToolName,
			Description: "Get total income, expenses, and net for any inclusive booking-date range (single day, month, or calendar year). Use only from and to — no limit parameter.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"from": {"type": "string", "description": "Start date (YYYY-MM-DD, inclusive). Must be on or before to."},
					"to": {"type": "string", "description": "End date (YYYY-MM-DD, inclusive). Example calendar year 2025: from=2025-01-01, to=2025-12-31."}
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
		OK:       true,
		From:     args.From,
		To:       args.To,
		Income:   decimalString(report.Income),
		Expenses: decimalString(report.Expenses),
		Net:      decimalString(report.Net),
		Currency: "EUR",
	}, nil
}
