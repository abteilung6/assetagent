package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/llm"
)

const (
	searchToolName      = "search_transactions"
	defaultSearchLimit  = 10
	maxSearchLimit      = 50
)

type searchArgs struct {
	Q     string  `json:"q"`
	From  *string `json:"from"`
	To    *string `json:"to"`
	Limit *int    `json:"limit"`
}

type searchResult struct {
	Total        int64                 `json:"total"`
	Transactions []searchTransaction   `json:"transactions"`
}

type searchTransaction struct {
	BookingDate  string `json:"booking_date"`
	Counterparty string `json:"counterparty"`
	Purpose      string `json:"purpose"`
	Amount       string `json:"amount"`
	Currency     string `json:"currency"`
}

func searchTool(lister TransactionLister) toolEntry {
	return toolEntry{
		definition: llm.Tool{
			Name: searchToolName,
			Description: "Search individual transactions by text in purpose, counterparty, or booking text. Requires q. Not for spending totals — use get_cashflow for how much was spent in a period.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"q": {"type": "string", "description": "Search text"},
					"from": {"type": "string", "description": "Optional start booking date (YYYY-MM-DD)"},
					"to": {"type": "string", "description": "Optional end booking date (YYYY-MM-DD)"},
					"limit": {"type": "integer", "description": "Maximum rows to return (default 10, max 50)"}
				},
				"required": ["q"]
			}`),
		},
		handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			return runSearch(ctx, lister, raw)
		},
	}
}

func runSearch(ctx context.Context, lister TransactionLister, raw json.RawMessage) (searchResult, error) {
	if lister == nil {
		return searchResult{}, errors.New("transaction lister is not configured")
	}

	var args searchArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return searchResult{}, fmt.Errorf("invalid arguments: %w", err)
	}
	if args.Q == "" {
		return searchResult{}, errors.New("q is required")
	}

	limit := defaultSearchLimit
	if args.Limit != nil {
		limit = *args.Limit
	}
	if limit <= 0 {
		return searchResult{}, errors.New("limit must be positive")
	}
	if limit > maxSearchLimit {
		return searchResult{}, fmt.Errorf("limit must be at most %d", maxSearchLimit)
	}

	params := domain.ListParams{
		Limit:  limit,
		Offset: 0,
		Search: &args.Q,
		Sort:   domain.SortBookingDate,
	}

	if args.From != nil && *args.From != "" {
		from, err := time.Parse("2006-01-02", *args.From)
		if err != nil {
			return searchResult{}, fmt.Errorf("invalid from date: %w", err)
		}
		params.FromDate = &from
	}
	if args.To != nil && *args.To != "" {
		to, err := time.Parse("2006-01-02", *args.To)
		if err != nil {
			return searchResult{}, fmt.Errorf("invalid to date: %w", err)
		}
		params.ToDate = &to
	}
	if params.FromDate != nil && params.ToDate != nil && params.ToDate.Before(*params.FromDate) {
		return searchResult{}, errors.New("to must be on or after from")
	}

	list, err := lister.List(ctx, params)
	if err != nil {
		return searchResult{}, err
	}

	result := searchResult{
		Total:        list.Total,
		Transactions: make([]searchTransaction, len(list.Transactions)),
	}
	for i, tx := range list.Transactions {
		result.Transactions[i] = searchTransaction{
			BookingDate:  tx.BookingDate.Format("2006-01-02"),
			Counterparty: tx.Counterparty,
			Purpose:      tx.Purpose,
			Amount:       decimalString(tx.Amount),
			Currency:     tx.Currency,
		}
	}

	return result, nil
}
