package evals

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/db"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type expectedFile struct {
	Period struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"period"`
	ConfirmTransfers bool `json:"confirm_transfers"`
	CashflowRaw      struct {
		Income   string `json:"income"`
		Expenses string `json:"expenses"`
		Net      string `json:"net"`
	} `json:"cashflow_raw"`
	CashflowV2 struct {
		Income   string `json:"income"`
		Expenses string `json:"expenses"`
		Net      string `json:"net"`
	} `json:"cashflow_v2"`
	Transfers struct {
		Confirmed int    `json:"confirmed"`
		Net       string `json:"net"`
	} `json:"transfers"`
	Recurring []struct {
		DisplayNameContains string `json:"display_name_contains"`
		Interval            string `json:"interval"`
		AmountTypical       string `json:"amount_typical"`
		MemberCount         int    `json:"member_count"`
	} `json:"recurring"`
}

func TestGoldenHouseholds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping golden suite")
	}

	root := filepath.Join("..", "..", "testdata", "golden")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read golden root: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		t.Run(name, func(t *testing.T) {
			runGoldenHousehold(t, filepath.Join(root, name))
		})
	}
}

func runGoldenHousehold(t *testing.T, dir string) {
	t.Helper()

	expectedPath := filepath.Join(dir, "expected.json")
	raw, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("read expected.json: %v", err)
	}
	var expected expectedFile
	if err := json.Unmarshal(raw, &expected); err != nil {
		t.Fatalf("parse expected.json: %v", err)
	}

	ctx := context.Background()
	pool := setupPostgres(t, ctx)
	t.Cleanup(pool.Close)

	importer := service.NewImport(pool)
	csvs, err := filepath.Glob(filepath.Join(dir, "*.csv"))
	if err != nil {
		t.Fatalf("glob csv: %v", err)
	}
	if len(csvs) == 0 {
		t.Fatal("no CSV fixtures")
	}
	for _, csvPath := range csvs {
		base := strings.TrimSuffix(filepath.Base(csvPath), ".csv")
		accountName := "Golden " + strings.ReplaceAll(base, "_", " ")
		if _, err := importer.ImportFile(ctx, csvPath, domain.ImportOptions{
			AccountName: accountName,
		}); err != nil {
			t.Fatalf("import %s: %v", csvPath, err)
		}
	}

	transfers := service.NewTransfers(pool)
	if _, err := transfers.Scan(ctx); err != nil {
		t.Fatalf("transfer scan: %v", err)
	}
	if expected.ConfirmTransfers {
		pairs, err := transfers.List(ctx)
		if err != nil {
			t.Fatalf("list transfers: %v", err)
		}
		for _, pair := range pairs {
			if pair.Status != domain.TransferStatusSuggested {
				continue
			}
			if _, err := transfers.Confirm(ctx, pair.ID); err != nil {
				t.Fatalf("confirm transfer %s: %v", pair.ID, err)
			}
		}
	}

	if _, err := service.NewClassify(pool).Run(ctx); err != nil {
		t.Fatalf("classify run: %v", err)
	}
	if _, err := service.NewRecurring(pool).Scan(ctx); err != nil {
		t.Fatalf("recurring scan: %v", err)
	}

	from := mustParseDate(t, expected.Period.From)
	to := mustParseDate(t, expected.Period.To)
	// GetCashflow uses [from, to) exclusive end in some repos — check reports.
	reports := repository.NewReports(pool)

	rawCF, err := reports.GetCashflow(ctx, from, to)
	if err != nil {
		t.Fatalf("cashflow raw: %v", err)
	}
	assertCashflow(t, "cashflow_raw", rawCF.Income, rawCF.Expenses, rawCF.Net,
		expected.CashflowRaw.Income, expected.CashflowRaw.Expenses, expected.CashflowRaw.Net)

	v2, err := reports.GetCashflowV2(ctx, from, to)
	if err != nil {
		t.Fatalf("cashflow v2: %v", err)
	}
	assertCashflow(t, "cashflow_v2", v2.Income, v2.Expenses, v2.Net,
		expected.CashflowV2.Income, expected.CashflowV2.Expenses, expected.CashflowV2.Net)

	confirmed := 0
	pairs, err := transfers.List(ctx)
	if err != nil {
		t.Fatalf("list transfers after confirm: %v", err)
	}
	for _, pair := range pairs {
		if pair.Status == domain.TransferStatusConfirmed {
			confirmed++
		}
	}
	if confirmed != expected.Transfers.Confirmed {
		t.Fatalf("confirmed transfers = %d, want %d", confirmed, expected.Transfers.Confirmed)
	}
	if expected.Transfers.Net != "0.00" {
		t.Fatalf("fixture transfers.net must be 0.00 for Phase B invariant, got %s", expected.Transfers.Net)
	}

	series, err := service.NewRecurring(pool).List(ctx)
	if err != nil {
		t.Fatalf("list recurring: %v", err)
	}
	for _, want := range expected.Recurring {
		found := false
		for _, got := range series {
			if !strings.Contains(got.DisplayName, want.DisplayNameContains) {
				continue
			}
			found = true
			if got.Interval != want.Interval {
				t.Fatalf("recurring %q interval = %q, want %q", got.DisplayName, got.Interval, want.Interval)
			}
			if !got.AmountTypical.Equal(mustDecimal(t, want.AmountTypical)) {
				t.Fatalf("recurring %q amount_typical = %s, want %s", got.DisplayName, got.AmountTypical, want.AmountTypical)
			}
			if got.MemberCount != want.MemberCount {
				t.Fatalf("recurring %q members = %d, want %d", got.DisplayName, got.MemberCount, want.MemberCount)
			}
			break
		}
		if !found {
			t.Fatalf("missing recurring series containing %q (have %d series)", want.DisplayNameContains, len(series))
		}
	}
}

func assertCashflow(
	t *testing.T,
	label string,
	income, expenses, net decimal.Decimal,
	wantIncome, wantExpenses, wantNet string,
) {
	t.Helper()
	wi := mustDecimal(t, wantIncome)
	we := mustDecimal(t, wantExpenses)
	wn := mustDecimal(t, wantNet)
	if !income.Equal(wi) || !expenses.Equal(we) || !net.Equal(wn) {
		t.Fatalf("%s = income=%s expenses=%s net=%s, want income=%s expenses=%s net=%s",
			label, income, expenses, net, wi, we, wn)
	}
}

func mustDecimal(t *testing.T, raw string) decimal.Decimal {
	t.Helper()
	v, err := decimal.NewFromString(raw)
	if err != nil {
		t.Fatalf("decimal %q: %v", raw, err)
	}
	return v
}

func mustParseDate(t *testing.T, raw string) time.Time {
	t.Helper()
	v, err := time.Parse("2006-01-02", raw)
	if err != nil {
		t.Fatalf("date %q: %v", raw, err)
	}
	return time.Date(v.Year(), v.Month(), v.Day(), 0, 0, 0, 0, time.UTC)
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

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		p, err := db.NewPool(ctx, connStr)
		if err == nil {
			if err := p.Ping(ctx); err == nil {
				p.Close()
				break
			}
			p.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}

	if err := db.RunMigrations(connStr, "up"); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	pool, err := db.NewPool(ctx, connStr)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	return pool
}
