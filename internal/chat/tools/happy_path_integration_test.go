package tools_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/chat/tools"
	"github.com/abteilung6/assetagent/internal/db"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// TestIntegration_ImportCashflowEvidence locks the pre-Phase-A demo path:
// Sparkasse CSV import → get_cashflow tool → search_transactions evidence.
func TestIntegration_ImportCashflowEvidence(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	t.Cleanup(pool.Close)

	txRepo := repository.NewTransaction(pool)
	importer := service.NewImport(txRepo)
	registry := tools.NewRegistry(tools.Dependencies{
		Reports: repository.NewReports(pool),
		Lister:  txRepo,
	})

	fixture := filepath.Join("..", "..", "..", "testdata", "sparkasse", "minimal.csv")
	result, err := importer.ImportFile(ctx, fixture)
	if err != nil {
		t.Fatalf("ImportFile(%s): %v", fixture, err)
	}
	if result.Rows != 6 || result.Inserted != 6 || result.Duplicates != 0 {
		t.Fatalf("import result = %+v, want rows/inserted=6 duplicates=0", result)
	}

	cashflowRaw, err := registry.Execute(ctx, "get_cashflow", json.RawMessage(`{
		"from": "2025-12-01",
		"to": "2025-12-31"
	}`))
	if err != nil {
		t.Fatalf("get_cashflow: %v", err)
	}

	var cashflow struct {
		OK       bool   `json:"ok"`
		Income   string `json:"income"`
		Expenses string `json:"expenses"`
		Net      string `json:"net"`
		Currency string `json:"currency"`
	}
	if err := json.Unmarshal(cashflowRaw, &cashflow); err != nil {
		t.Fatalf("decode cashflow: %v", err)
	}
	if !cashflow.OK {
		t.Fatalf("cashflow ok = false, body = %s", cashflowRaw)
	}
	if cashflow.Currency != "EUR" {
		t.Fatalf("currency = %q, want EUR", cashflow.Currency)
	}
	// minimal.csv Dec 2025: income 56000+100, expenses 23.97+2.99+4.95+11.50
	if cashflow.Income != "56100" && cashflow.Income != "56100.00" {
		t.Fatalf("income = %q, want 56100", cashflow.Income)
	}
	if cashflow.Expenses != "43.41" {
		t.Fatalf("expenses = %q, want 43.41", cashflow.Expenses)
	}
	if cashflow.Net != "56056.59" {
		t.Fatalf("net = %q, want 56056.59", cashflow.Net)
	}

	searchRaw, err := registry.Execute(ctx, "search_transactions", json.RawMessage(`{
		"q": "AMAZON",
		"from": "2025-12-01",
		"to": "2025-12-31",
		"limit": 10
	}`))
	if err != nil {
		t.Fatalf("search_transactions: %v", err)
	}

	var search struct {
		OK           bool  `json:"ok"`
		Total        int64 `json:"total"`
		Transactions []struct {
			Counterparty string `json:"counterparty"`
			Purpose      string `json:"purpose"`
			Amount       string `json:"amount"`
		} `json:"transactions"`
	}
	if err := json.Unmarshal(searchRaw, &search); err != nil {
		t.Fatalf("decode search: %v", err)
	}
	if !search.OK {
		t.Fatalf("search ok = false, body = %s", searchRaw)
	}
	if search.Total < 1 || len(search.Transactions) < 1 {
		t.Fatalf("search evidence empty: total=%d txs=%d body=%s", search.Total, len(search.Transactions), searchRaw)
	}
	foundAmazon := false
	for _, tx := range search.Transactions {
		if strings.Contains(strings.ToUpper(tx.Counterparty), "AMAZON") ||
			strings.Contains(strings.ToUpper(tx.Purpose), "PRIME") {
			foundAmazon = true
			if tx.Amount != "-2.99" {
				t.Fatalf("amazon amount = %q, want -2.99", tx.Amount)
			}
			break
		}
	}
	if !foundAmazon {
		t.Fatalf("expected AMAZON evidence in search results: %s", searchRaw)
	}
}

func setupPostgres(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	container, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("assetagent"),
		postgres.WithUsername("assetagent"),
		postgres.WithPassword("assetagent"),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("terminate postgres: %v", err)
		}
	})

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}

	waitForDB(ctx, t, connStr)

	if err := db.RunMigrations(connStr, "up"); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	pool, err := db.NewPool(ctx, connStr)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}

	return pool
}

func waitForDB(ctx context.Context, t *testing.T, connStr string) {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		pool, err := db.NewPool(ctx, connStr)
		if err == nil {
			if err := pool.Ping(ctx); err == nil {
				pool.Close()
				return
			}
			pool.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Fatal("database not ready")
}
