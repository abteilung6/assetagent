package tools_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/chat/tools"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/shopspring/decimal"
)

type fakeReports struct {
	cashflow       domain.CashflowReport
	counterparties []domain.CounterpartySpend
	cashflowFrom   time.Time
	cashflowTo     time.Time
	counterFrom    time.Time
	counterTo      time.Time
	counterLimit   int
}

func (f *fakeReports) GetCashflow(ctx context.Context, from, to time.Time) (domain.CashflowReport, error) {
	f.cashflowFrom = from
	f.cashflowTo = to
	return f.cashflow, nil
}

func (f *fakeReports) GetTopCounterparties(
	ctx context.Context,
	from, to time.Time,
	limit int,
) ([]domain.CounterpartySpend, error) {
	f.counterFrom = from
	f.counterTo = to
	f.counterLimit = limit
	return f.counterparties, nil
}

type fakeLister struct {
	params domain.ListParams
	result domain.ListResult
	err    error
}

func (f *fakeLister) List(ctx context.Context, params domain.ListParams) (domain.ListResult, error) {
	f.params = params
	return f.result, f.err
}

func TestRegistry_tools(t *testing.T) {
	registry := tools.NewRegistry(tools.Dependencies{
		Reports: &fakeReports{},
		Lister:  &fakeLister{},
	})

	names := make(map[string]bool)
	for _, tool := range registry.Tools() {
		if tool.Name == "" || tool.Description == "" || len(tool.Parameters) == 0 {
			t.Fatalf("invalid tool definition: %+v", tool)
		}
		names[tool.Name] = true
	}

	for _, want := range []string{
		"get_cashflow",
		"get_top_counterparties",
		"search_transactions",
	} {
		if !names[want] {
			t.Fatalf("missing tool %q", want)
		}
	}
}

func TestRegistry_executeCashflow(t *testing.T) {
	reports := &fakeReports{
		cashflow: domain.CashflowReport{
			Income:   decimal.RequireFromString("5200.00"),
			Expenses: decimal.RequireFromString("2143.22"),
			Net:      decimal.RequireFromString("3056.78"),
		},
	}
	registry := tools.NewRegistry(tools.Dependencies{Reports: reports, Lister: &fakeLister{}})

	raw, err := registry.Execute(context.Background(), "get_cashflow", json.RawMessage(`{
		"from": "2025-06-01",
		"to": "2025-06-30"
	}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	var result struct {
		Income   string `json:"income"`
		Expenses string `json:"expenses"`
		Net      string `json:"net"`
		Currency string `json:"currency"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Income != "5200" && result.Income != "5200.00" {
		t.Fatalf("income = %q", result.Income)
	}
	if result.Expenses != "2143.22" {
		t.Fatalf("expenses = %q", result.Expenses)
	}
	if result.Net != "3056.78" {
		t.Fatalf("net = %q", result.Net)
	}
	if result.Currency != "EUR" {
		t.Fatalf("currency = %q", result.Currency)
	}
	if !reports.cashflowFrom.Equal(time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("from = %v", reports.cashflowFrom)
	}
	if !reports.cashflowTo.Equal(time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("to = %v", reports.cashflowTo)
	}
}

func TestRegistry_executeCounterparties(t *testing.T) {
	reports := &fakeReports{
		counterparties: []domain.CounterpartySpend{{
			Counterparty:     "REWE Markt GmbH",
			TotalSpent:       decimal.RequireFromString("153.40"),
			TransactionCount: 4,
		}},
	}
	registry := tools.NewRegistry(tools.Dependencies{Reports: reports, Lister: &fakeLister{}})

	raw, err := registry.Execute(context.Background(), "get_top_counterparties", json.RawMessage(`{
		"from": "2025-12-01",
		"to": "2025-12-31",
		"limit": 3
	}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	var result struct {
		Counterparties []struct {
			Counterparty     string `json:"counterparty"`
			TotalSpent       string `json:"total_spent"`
			TransactionCount int64  `json:"transaction_count"`
		} `json:"counterparties"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result.Counterparties) != 1 {
		t.Fatalf("counterparties = %+v", result.Counterparties)
	}
	if result.Counterparties[0].Counterparty != "REWE Markt GmbH" {
		t.Fatalf("counterparty = %q", result.Counterparties[0].Counterparty)
	}
	if reports.counterLimit != 3 {
		t.Fatalf("limit = %d, want 3", reports.counterLimit)
	}
}

func TestRegistry_executeCounterparties_stringLimit(t *testing.T) {
	reports := &fakeReports{
		counterparties: []domain.CounterpartySpend{{
			Counterparty:     "REWE Markt GmbH",
			TotalSpent:       decimal.RequireFromString("153.40"),
			TransactionCount: 4,
		}},
	}
	registry := tools.NewRegistry(tools.Dependencies{Reports: reports, Lister: &fakeLister{}})

	_, err := registry.Execute(context.Background(), "get_top_counterparties", json.RawMessage(`{
		"from": "2025-06-01",
		"to": "2025-06-30",
		"limit": "5"
	}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if reports.counterLimit != 5 {
		t.Fatalf("limit = %d, want 5", reports.counterLimit)
	}
}

func TestRegistry_executeSearch(t *testing.T) {
	lister := &fakeLister{
		result: domain.ListResult{
			Total: 1,
			Transactions: []domain.Transaction{{
				BookingDate:  time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC),
				Counterparty: "AMAZON DIGITAL GERMANY GMBH",
				Purpose:      "Prime Video",
				Amount:       decimal.RequireFromString("-2.99"),
				Currency:     "EUR",
			}},
		},
	}
	registry := tools.NewRegistry(tools.Dependencies{Reports: &fakeReports{}, Lister: lister})

	raw, err := registry.Execute(context.Background(), "search_transactions", json.RawMessage(`{
		"q": "Prime",
		"from": "2025-12-01",
		"to": "2025-12-31",
		"limit": 5
	}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	var result struct {
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("total = %d, want 1", result.Total)
	}
	if lister.params.Search == nil || *lister.params.Search != "Prime" {
		t.Fatalf("search = %v", lister.params.Search)
	}
	if lister.params.Limit != 5 {
		t.Fatalf("limit = %d, want 5", lister.params.Limit)
	}
	if lister.params.Sort != domain.SortBookingDate {
		t.Fatalf("sort = %q", lister.params.Sort)
	}
}

func TestRegistry_unknownTool(t *testing.T) {
	registry := tools.NewRegistry(tools.Dependencies{
		Reports: &fakeReports{},
		Lister:  &fakeLister{},
	})

	_, err := registry.Execute(context.Background(), "missing_tool", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, tools.ErrUnknownTool) {
		t.Fatalf("error = %v, want ErrUnknownTool", err)
	}
}

func TestRegistry_invalidCashflowDates(t *testing.T) {
	registry := tools.NewRegistry(tools.Dependencies{
		Reports: &fakeReports{},
		Lister:  &fakeLister{},
	})

	_, err := registry.Execute(context.Background(), "get_cashflow", json.RawMessage(`{
		"from": "2025-12-31",
		"to": "2025-12-01"
	}`))
	if err == nil {
		t.Fatal("expected error for reversed date range")
	}
}
