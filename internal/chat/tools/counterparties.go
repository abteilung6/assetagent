package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/abteilung6/assetagent/internal/llm"
)

const (
	counterpartiesToolName = "get_top_counterparties"
	defaultCounterpartyLimit = 5
	maxCounterpartyLimit     = 20
)

type counterpartiesArgs struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Limit *int   `json:"limit"`
}

type counterpartiesResult struct {
	Counterparties []counterpartySpendResult `json:"counterparties"`
}

type counterpartySpendResult struct {
	Counterparty     string `json:"counterparty"`
	TotalSpent       string `json:"total_spent"`
	TransactionCount int64  `json:"transaction_count"`
	Currency         string `json:"currency"`
}

func counterpartiesTool(reports Reports) toolEntry {
	return toolEntry{
		definition: llm.Tool{
			Name: counterpartiesToolName,
			Description: "Get the top counterparties by total spending in a booking-date range. Optional limit caps how many names are returned (max 20).",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"from": {"type": "string", "description": "Start date (YYYY-MM-DD, inclusive)"},
					"to": {"type": "string", "description": "End date (YYYY-MM-DD, inclusive)"},
					"limit": {"type": "integer", "description": "Maximum number of counterparties to return (default 5, max 20)"}
				},
				"required": ["from", "to"]
			}`),
		},
		handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			return runCounterparties(ctx, reports, raw)
		},
	}
}

func runCounterparties(ctx context.Context, reports Reports, raw json.RawMessage) (counterpartiesResult, error) {
	if reports == nil {
		return counterpartiesResult{}, errors.New("reports repository is not configured")
	}

	var args counterpartiesArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return counterpartiesResult{}, fmt.Errorf("invalid arguments: %w", err)
	}

	from, to, err := parseDateRange(args.From, args.To)
	if err != nil {
		return counterpartiesResult{}, err
	}

	limit := defaultCounterpartyLimit
	if args.Limit != nil {
		limit = *args.Limit
	}
	if limit <= 0 {
		return counterpartiesResult{}, errors.New("limit must be positive")
	}
	if limit > maxCounterpartyLimit {
		return counterpartiesResult{}, fmt.Errorf("limit must be at most %d", maxCounterpartyLimit)
	}

	spends, err := reports.GetTopCounterparties(ctx, from, to, limit)
	if err != nil {
		return counterpartiesResult{}, err
	}

	result := counterpartiesResult{
		Counterparties: make([]counterpartySpendResult, len(spends)),
	}
	for i, spend := range spends {
		result.Counterparties[i] = counterpartySpendResult{
			Counterparty:     spend.Counterparty,
			TotalSpent:       decimalString(spend.TotalSpent),
			TransactionCount: spend.TransactionCount,
			Currency:         "EUR",
		}
	}

	return result, nil
}
