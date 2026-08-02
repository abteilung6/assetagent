package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/abteilung6/assetagent/internal/llm"
)

const cashflowV2ToolName = "get_cashflow_v2"

type cashflowV2Result struct {
	OK                bool     `json:"ok"`
	Income            string   `json:"income"`
	Expenses          string   `json:"expenses"`
	Net               string   `json:"net"`
	Currency          string   `json:"currency"`
	Period            period   `json:"period"`
	AccountsIncluded  []string `json:"accounts_included"`
	TransfersExcluded bool     `json:"transfers_excluded"`
	Calculation       string   `json:"calculation"`
	Confidence        string   `json:"confidence"`
	DataFreshness     string   `json:"data_freshness"`
	Assumptions       []string `json:"assumptions"`
	EvidenceIDs       []string `json:"evidence_ids"`
}

type period struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func cashflowV2Tool(reports Reports) toolEntry {
	return toolEntry{
		definition: llm.Tool{
			Name: cashflowV2ToolName,
			Description: "Get household income, expenses, and net for a booking-date range with confirmed internal transfers excluded. Prefer this over get_cashflow for trusted money questions. Returns an evidence contract (accounts, calculation, confidence, evidence_ids).",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {
					"from": {"type": "string", "description": "Start date (YYYY-MM-DD, inclusive). Must be on or before to."},
					"to": {"type": "string", "description": "End date (YYYY-MM-DD, inclusive)."}
				},
				"required": ["from", "to"]
			}`),
		},
		handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			return runCashflowV2(ctx, reports, raw)
		},
	}
}

func runCashflowV2(ctx context.Context, reports Reports, raw json.RawMessage) (cashflowV2Result, error) {
	if reports == nil {
		return cashflowV2Result{}, errors.New("reports repository is not configured")
	}

	var args cashflowArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return cashflowV2Result{}, fmt.Errorf("invalid arguments: %w", err)
	}

	from, to, err := parseDateRange(args.From, args.To)
	if err != nil {
		return cashflowV2Result{}, err
	}

	evidence, err := reports.GetCashflowV2Evidence(ctx, from, to)
	if err != nil {
		return cashflowV2Result{}, err
	}

	accounts := evidence.AccountsIncluded
	if accounts == nil {
		accounts = []string{}
	}
	assumptions := evidence.Assumptions
	if assumptions == nil {
		assumptions = []string{}
	}
	ids := evidence.EvidenceIDs
	if ids == nil {
		ids = []string{}
	}

	return cashflowV2Result{
		OK:       true,
		Income:   decimalString(evidence.Income),
		Expenses: decimalString(evidence.Expenses),
		Net:      decimalString(evidence.Net),
		Currency: evidence.Currency,
		Period: period{
			From: args.From,
			To:   args.To,
		},
		AccountsIncluded:  accounts,
		TransfersExcluded: evidence.TransfersExcluded,
		Calculation:       evidence.Calculation,
		Confidence:        evidence.Confidence,
		DataFreshness:     evidence.DataFreshness,
		Assumptions:       assumptions,
		EvidenceIDs:       ids,
	}, nil
}
