package tools_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/chat/tools"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/shopspring/decimal"
)

type insightReports struct {
	cashflowV2 domain.CashflowV2Evidence
}

func (f *insightReports) GetCashflow(ctx context.Context, from, to time.Time) (domain.CashflowReport, error) {
	return domain.CashflowReport{}, nil
}

func (f *insightReports) GetCashflowV2Evidence(
	ctx context.Context,
	from, to time.Time,
) (domain.CashflowV2Evidence, error) {
	return f.cashflowV2, nil
}

func (f *insightReports) GetTopCounterparties(
	ctx context.Context,
	from, to time.Time,
	limit int,
) ([]domain.CounterpartySpend, error) {
	return nil, nil
}

type insightLister struct{}

func (f *insightLister) List(ctx context.Context, params domain.ListParams) (domain.ListResult, error) {
	return domain.ListResult{}, nil
}

type fakeInsights struct {
	spend  []repository.CategorySpendPoint
	impact repository.OneOffExpenseImpact
}

func (f *fakeInsights) CategorySpend(ctx context.Context, from, to time.Time, limit int) ([]repository.CategorySpendPoint, error) {
	return f.spend, nil
}

func (f *fakeInsights) OneOffImpact(ctx context.Context, from, to time.Time) (repository.OneOffExpenseImpact, error) {
	return f.impact, nil
}

func TestMonthCashflowTool_yyyyMM(t *testing.T) {
	reports := &insightReports{
		cashflowV2: domain.CashflowV2Evidence{
			Income:            decimal.NewFromInt(3000),
			Expenses:          decimal.NewFromInt(2000),
			Net:               decimal.NewFromInt(1000),
			Currency:          "EUR",
			TransfersExcluded: true,
			Calculation:       "cashflow_v2",
			Confidence:        "high",
			EvidenceIDs:       []string{"e1"},
		},
	}
	registry := tools.NewRegistry(tools.Dependencies{Reports: reports, Lister: &insightLister{}})
	raw, err := registry.Execute(context.Background(), "get_month_cashflow", json.RawMessage(`{"yyyy_mm":"2026-03"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result["ok"] != true || result["yyyy_mm"] != "2026-03" || result["deep_link"] != "/insights/months/2026-03" {
		t.Fatalf("result = %+v", result)
	}
	period, _ := result["period"].(map[string]any)
	if period["from"] != "2026-03-01" || period["to"] != "2026-03-31" {
		t.Fatalf("period = %+v", period)
	}
}

func TestCategorySpendAndOneOffTools(t *testing.T) {
	insights := &fakeInsights{
		spend: []repository.CategorySpendPoint{{
			CategorySlug:     "groceries",
			CategoryName:     "Groceries",
			Total:            decimal.NewFromInt(400),
			TransactionCount: 12,
		}},
		impact: repository.OneOffExpenseImpact{
			Count:        2,
			ExpenseTotal: decimal.NewFromInt(500),
		},
	}
	registry := tools.NewRegistry(tools.Dependencies{
		Reports:  &insightReports{},
		Lister:   &insightLister{},
		Insights: insights,
	})

	spendRaw, err := registry.Execute(context.Background(), "get_category_spend", json.RawMessage(`{"from":"2026-03-01","to":"2026-03-31"}`))
	if err != nil {
		t.Fatalf("category spend: %v", err)
	}
	var spend map[string]any
	if err := json.Unmarshal(spendRaw, &spend); err != nil {
		t.Fatalf("unmarshal spend: %v", err)
	}
	items, _ := spend["items"].([]any)
	if spend["ok"] != true || len(items) != 1 || spend["deep_link"] != "/insights/months/2026-03" {
		t.Fatalf("spend = %+v", spend)
	}

	impactRaw, err := registry.Execute(context.Background(), "get_one_off_impact", json.RawMessage(`{"from":"2026-03-01","to":"2026-03-31"}`))
	if err != nil {
		t.Fatalf("one-off: %v", err)
	}
	var impact map[string]any
	if err := json.Unmarshal(impactRaw, &impact); err != nil {
		t.Fatalf("unmarshal impact: %v", err)
	}
	if impact["ok"] != true || impact["count"] != float64(2) || impact["expense_total"] != "500" {
		t.Fatalf("impact = %+v", impact)
	}
}
