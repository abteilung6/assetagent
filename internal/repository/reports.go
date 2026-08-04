package repository

import (
	"context"
	"fmt"
	"time"

	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type Reports struct {
	pool    *pgxpool.Pool
	queries sqldb.Querier
}

func NewReports(pool *pgxpool.Pool) *Reports {
	return &Reports{pool: pool, queries: sqldb.New(pool)}
}

func (r *Reports) GetCashflow(
	ctx context.Context,
	from, to time.Time,
) (domain.CashflowReport, error) {
	householdID, err := ResolveHouseholdID(ctx, r.pool)
	if err != nil {
		return domain.CashflowReport{}, err
	}
	row, err := r.queries.GetCashflow(ctx, sqldb.GetCashflowParams{
		HouseholdID: householdID,
		FromDate:    pgtype.Date{Time: from, Valid: true},
		ToDate:      pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return domain.CashflowReport{}, err
	}

	return domain.CashflowReport{
		Income:   row.Income,
		Expenses: row.Expenses,
		Net:      row.Net,
	}, nil
}

func (r *Reports) GetCashflowV2(
	ctx context.Context,
	from, to time.Time,
) (domain.CashflowReportV2, error) {
	householdID, err := ResolveHouseholdID(ctx, r.pool)
	if err != nil {
		return domain.CashflowReportV2{}, err
	}
	row, err := r.queries.GetCashflowV2(ctx, sqldb.GetCashflowV2Params{
		HouseholdID: householdID,
		FromDate:    pgtype.Date{Time: from, Valid: true},
		ToDate:      pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return domain.CashflowReportV2{}, err
	}

	return domain.CashflowReportV2{
		Income:            row.Income,
		Expenses:          row.Expenses,
		Net:               row.Net,
		TransfersExcluded: true,
	}, nil
}

func (r *Reports) GetCashflowV2Evidence(
	ctx context.Context,
	from, to time.Time,
) (domain.CashflowV2Evidence, error) {
	householdID, err := ResolveHouseholdID(ctx, r.pool)
	if err != nil {
		return domain.CashflowV2Evidence{}, err
	}

	report, err := r.GetCashflowV2(ctx, from, to)
	if err != nil {
		return domain.CashflowV2Evidence{}, err
	}

	fromDate := pgtype.Date{Time: from, Valid: true}
	toDate := pgtype.Date{Time: to, Valid: true}

	accounts, err := r.queries.ListAccountsInPeriod(ctx, sqldb.ListAccountsInPeriodParams{
		HouseholdID: householdID,
		FromDate:    fromDate,
		ToDate:      toDate,
	})
	if err != nil {
		return domain.CashflowV2Evidence{}, fmt.Errorf("list accounts: %w", err)
	}

	transferIDs, err := r.queries.ListConfirmedTransferIDsInPeriod(ctx, sqldb.ListConfirmedTransferIDsInPeriodParams{
		HouseholdID: householdID,
		FromDate:    fromDate,
		ToDate:      toDate,
	})
	if err != nil {
		return domain.CashflowV2Evidence{}, fmt.Errorf("list transfers: %w", err)
	}

	txIDs, err := r.queries.ListCashflowV2TransactionIDs(ctx, sqldb.ListCashflowV2TransactionIDsParams{
		HouseholdID: householdID,
		FromDate:    fromDate,
		ToDate:      toDate,
		RowLimit:    50,
	})
	if err != nil {
		return domain.CashflowV2Evidence{}, fmt.Errorf("list evidence txs: %w", err)
	}

	freshness := ""
	latest, err := r.queries.GetLatestBookingDate(ctx, householdID)
	if err == nil && latest.Valid {
		freshness = latest.Time.Format("2006-01-02")
	}

	evidenceIDs := make([]string, 0, len(transferIDs)+len(txIDs))
	for _, id := range transferIDs {
		evidenceIDs = append(evidenceIDs, "transfer_"+id.String())
	}
	for _, id := range txIDs {
		evidenceIDs = append(evidenceIDs, "tx_"+id.String())
	}

	confidence := "high"
	if report.Income.IsZero() && report.Expenses.IsZero() {
		confidence = "medium"
	}

	return domain.CashflowV2Evidence{
		Income:            report.Income,
		Expenses:          report.Expenses,
		Net:               report.Net,
		Currency:          "EUR",
		PeriodFrom:        from,
		PeriodTo:          to,
		AccountsIncluded:  accounts,
		TransfersExcluded: true,
		Calculation:       "Sum of booking amounts in the period, excluding legs of confirmed internal transfers and user-marked one-offs",
		Confidence:        confidence,
		DataFreshness:     freshness,
		Assumptions: []string{
			"Only confirmed internal transfers are excluded",
			"One-off transactions marked by the user are excluded",
			"Suggested or rejected transfer pairs still count as income/expense",
		},
		EvidenceIDs: evidenceIDs,
	}, nil
}

func (r *Reports) GetTopCounterparties(
	ctx context.Context,
	from, to time.Time,
	limit int,
) ([]domain.CounterpartySpend, error) {
	householdID, err := ResolveHouseholdID(ctx, r.pool)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.GetTopCounterparties(ctx, sqldb.GetTopCounterpartiesParams{
		HouseholdID: householdID,
		FromDate:    pgtype.Date{Time: from, Valid: true},
		ToDate:      pgtype.Date{Time: to, Valid: true},
		RowLimit:    int32(limit),
	})
	if err != nil {
		return nil, err
	}

	spends := make([]domain.CounterpartySpend, len(rows))
	for i, row := range rows {
		spends[i] = domain.CounterpartySpend{
			Counterparty:     row.Counterparty,
			TotalSpent:       row.TotalSpent,
			TransactionCount: row.TransactionCount,
		}
	}

	return spends, nil
}

// MonthlyCashflowV2 is one calendar month of transfer-aware cashflow.
type MonthlyCashflowV2 struct {
	MonthStart time.Time
	Income     decimal.Decimal
	Expenses   decimal.Decimal
	Net        decimal.Decimal
}

func (r *Reports) ListMonthlyCashflowV2(
	ctx context.Context,
	from, to time.Time,
) ([]MonthlyCashflowV2, error) {
	householdID, err := ResolveHouseholdID(ctx, r.pool)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.ListMonthlyCashflowV2(ctx, sqldb.ListMonthlyCashflowV2Params{
		HouseholdID: householdID,
		FromDate:    pgtype.Date{Time: from, Valid: true},
		ToDate:      pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	out := make([]MonthlyCashflowV2, 0, len(rows))
	for _, row := range rows {
		if !row.MonthStart.Valid {
			continue
		}
		out = append(out, MonthlyCashflowV2{
			MonthStart: row.MonthStart.Time,
			Income:     row.Income,
			Expenses:   row.Expenses,
			Net:        row.Net,
		})
	}
	return out, nil
}

// OneOffExpenseImpact is excluded spend that still shapes trust copy.
type OneOffExpenseImpact struct {
	Count        int64
	ExpenseTotal decimal.Decimal
}

func (r *Reports) GetOneOffExpenseImpact(
	ctx context.Context,
	from, to time.Time,
) (OneOffExpenseImpact, error) {
	householdID, err := ResolveHouseholdID(ctx, r.pool)
	if err != nil {
		return OneOffExpenseImpact{}, err
	}
	row, err := r.queries.GetOneOffExpenseImpact(ctx, sqldb.GetOneOffExpenseImpactParams{
		HouseholdID: householdID,
		FromDate:    pgtype.Date{Time: from, Valid: true},
		ToDate:      pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return OneOffExpenseImpact{}, err
	}
	return OneOffExpenseImpact{
		Count:        row.OneOffCount,
		ExpenseTotal: row.OneOffExpenseTotal,
	}, nil
}

// CategorySpendPoint is expense total for one category in a period.
type CategorySpendPoint struct {
	CategorySlug     string
	CategoryName     string
	Total            decimal.Decimal
	TransactionCount int64
}

func (r *Reports) ListCategorySpend(
	ctx context.Context,
	from, to time.Time,
	limit int,
) ([]CategorySpendPoint, error) {
	householdID, err := ResolveHouseholdID(ctx, r.pool)
	if err != nil {
		return nil, err
	}
	if limit < 1 {
		limit = 8
	}
	if limit > 50 {
		limit = 50
	}
	rows, err := r.queries.ListCategorySpendInPeriod(ctx, sqldb.ListCategorySpendInPeriodParams{
		HouseholdID: householdID,
		FromDate:    pgtype.Date{Time: from, Valid: true},
		ToDate:      pgtype.Date{Time: to, Valid: true},
		RowLimit:    int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]CategorySpendPoint, 0, len(rows))
	for _, row := range rows {
		out = append(out, CategorySpendPoint{
			CategorySlug:     row.CategorySlug,
			CategoryName:     row.CategoryName,
			Total:            row.Total,
			TransactionCount: row.TransactionCount,
		})
	}
	return out, nil
}

// CategoryMerchantSpendPoint is expense total for one merchant within a category.
type CategoryMerchantSpendPoint struct {
	Merchant         string
	Total            decimal.Decimal
	TransactionCount int64
}

func (r *Reports) ListMerchantSpendInCategory(
	ctx context.Context,
	from, to time.Time,
	categorySlug string,
	limit int,
) ([]CategoryMerchantSpendPoint, error) {
	householdID, err := ResolveHouseholdID(ctx, r.pool)
	if err != nil {
		return nil, err
	}
	if limit < 1 {
		limit = 8
	}
	if limit > 20 {
		limit = 20
	}
	rows, err := r.queries.ListMerchantSpendInCategoryPeriod(ctx, sqldb.ListMerchantSpendInCategoryPeriodParams{
		HouseholdID:  householdID,
		FromDate:     pgtype.Date{Time: from, Valid: true},
		ToDate:       pgtype.Date{Time: to, Valid: true},
		CategorySlug: categorySlug,
		RowLimit:     int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]CategoryMerchantSpendPoint, 0, len(rows))
	for _, row := range rows {
		out = append(out, CategoryMerchantSpendPoint{
			Merchant:         row.Merchant,
			Total:            row.Total,
			TransactionCount: row.TransactionCount,
		})
	}
	return out, nil
}

// MonthlyCategorySpendPoint is expense total for one category in one calendar month.
type MonthlyCategorySpendPoint struct {
	MonthStart   time.Time
	CategorySlug string
	CategoryName string
	Total        decimal.Decimal
}

func (r *Reports) ListMonthlyCategorySpend(
	ctx context.Context,
	from, to time.Time,
	categoryLimit int,
) ([]MonthlyCategorySpendPoint, error) {
	householdID, err := ResolveHouseholdID(ctx, r.pool)
	if err != nil {
		return nil, err
	}
	if categoryLimit < 1 {
		categoryLimit = 5
	}
	if categoryLimit > 12 {
		categoryLimit = 12
	}
	rows, err := r.queries.ListMonthlyCategorySpendInPeriod(ctx, sqldb.ListMonthlyCategorySpendInPeriodParams{
		HouseholdID:   householdID,
		FromDate:      pgtype.Date{Time: from, Valid: true},
		ToDate:        pgtype.Date{Time: to, Valid: true},
		CategoryLimit: int32(categoryLimit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]MonthlyCategorySpendPoint, 0, len(rows))
	for _, row := range rows {
		out = append(out, MonthlyCategorySpendPoint{
			MonthStart:   row.MonthStart.Time,
			CategorySlug: row.CategorySlug,
			CategoryName: row.CategoryName,
			Total:        row.Total,
		})
	}
	return out, nil
}

// DailyExpensePacePoint is expense total and booking count for one day.
type DailyExpensePacePoint struct {
	Date             time.Time
	Expenses         decimal.Decimal
	TransactionCount int64
}

func (r *Reports) ListDailyExpensePace(
	ctx context.Context,
	from, to time.Time,
) ([]DailyExpensePacePoint, error) {
	householdID, err := ResolveHouseholdID(ctx, r.pool)
	if err != nil {
		return nil, err
	}
	rows, err := r.queries.ListDailyExpensePaceInPeriod(ctx, sqldb.ListDailyExpensePaceInPeriodParams{
		HouseholdID: householdID,
		FromDate:    pgtype.Date{Time: from, Valid: true},
		ToDate:      pgtype.Date{Time: to, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	out := make([]DailyExpensePacePoint, 0, len(rows))
	for _, row := range rows {
		if !row.BookingDay.Valid {
			continue
		}
		out = append(out, DailyExpensePacePoint{
			Date:             row.BookingDay.Time,
			Expenses:         row.Expenses,
			TransactionCount: row.TransactionCount,
		})
	}
	return out, nil
}
