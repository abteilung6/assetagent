package tools_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/abteilung6/assetagent/internal/chat/tools"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/shopspring/decimal"
)

func TestIntegration_GetCashflowV2GoldenCoupleTransfer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	t.Cleanup(pool.Close)

	importer := service.NewImport(pool)
	dir := filepath.Join("..", "..", "..", "testdata", "golden", "couple_transfer")
	for _, name := range []string{"checking.csv", "savings.csv"} {
		path := filepath.Join(dir, name)
		if _, err := importer.ImportFile(ctx, path, domain.ImportOptions{
			AccountName: "Golden " + name,
		}); err != nil {
			t.Fatalf("import %s: %v", path, err)
		}
	}

	transfers := service.NewTransfers(pool)
	if _, err := transfers.Scan(ctx); err != nil {
		t.Fatalf("scan: %v", err)
	}
	pairs, err := transfers.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, pair := range pairs {
		if pair.Status != domain.TransferStatusSuggested {
			continue
		}
		if _, err := transfers.Confirm(ctx, pair.ID); err != nil {
			t.Fatalf("confirm: %v", err)
		}
	}

	registry := tools.NewRegistry(tools.Dependencies{
		Reports: repository.NewReports(pool),
		Lister:  repository.NewTransaction(pool),
	})

	raw, err := registry.Execute(ctx, "get_cashflow_v2", json.RawMessage(`{
		"from": "2026-03-01",
		"to": "2026-03-31"
	}`))
	if err != nil {
		t.Fatalf("get_cashflow_v2: %v", err)
	}

	var result struct {
		OK                bool     `json:"ok"`
		Income            string   `json:"income"`
		Expenses          string   `json:"expenses"`
		Net               string   `json:"net"`
		Currency          string   `json:"currency"`
		TransfersExcluded bool     `json:"transfers_excluded"`
		EvidenceIDs       []string `json:"evidence_ids"`
		AccountsIncluded  []string `json:"accounts_included"`
		Calculation       string   `json:"calculation"`
		Confidence        string   `json:"confidence"`
		DataFreshness     string   `json:"data_freshness"`
		Assumptions       []string `json:"assumptions"`
		Period            struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"period"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !result.OK || !result.TransfersExcluded {
		t.Fatalf("result = %+v body=%s", result, raw)
	}
	assertDecimal(t, "income", result.Income, "2000.00")
	assertDecimal(t, "expenses", result.Expenses, "12.50")
	assertDecimal(t, "net", result.Net, "1987.50")
	if result.Currency != "EUR" {
		t.Fatalf("currency = %q", result.Currency)
	}
	if result.Period.From != "2026-03-01" || result.Period.To != "2026-03-31" {
		t.Fatalf("period = %+v", result.Period)
	}
	if result.Calculation == "" || result.Confidence == "" || result.DataFreshness == "" {
		t.Fatalf("missing evidence contract fields: %+v", result)
	}
	if len(result.Assumptions) == 0 || len(result.AccountsIncluded) == 0 {
		t.Fatalf("missing assumptions/accounts: %+v", result)
	}
	hasTransfer := false
	hasTx := false
	for _, id := range result.EvidenceIDs {
		if len(id) > 9 && id[:9] == "transfer_" {
			hasTransfer = true
		}
		if len(id) > 3 && id[:3] == "tx_" {
			hasTx = true
		}
	}
	if !hasTransfer || !hasTx {
		t.Fatalf("evidence_ids = %v, want transfer_ and tx_ prefixes", result.EvidenceIDs)
	}
}

func assertDecimal(t *testing.T, label, got, want string) {
	t.Helper()
	g, err := decimal.NewFromString(got)
	if err != nil {
		t.Fatalf("%s parse %q: %v", label, got, err)
	}
	w := decimal.RequireFromString(want)
	if !g.Equal(w) {
		t.Fatalf("%s = %s, want %s", label, got, want)
	}
}
