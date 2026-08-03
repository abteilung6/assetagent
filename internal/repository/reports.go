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
	queries sqldb.Querier
}

func NewReports(pool *pgxpool.Pool) *Reports {
	return &Reports{queries: sqldb.New(pool)}
}

func (r *Reports) GetCashflow(
	ctx context.Context,
	from, to time.Time,
) (domain.CashflowReport, error) {
	row, err := r.queries.GetCashflow(ctx, sqldb.GetCashflowParams{
		FromDate: pgtype.Date{Time: from, Valid: true},
		ToDate:   pgtype.Date{Time: to, Valid: true},
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
	row, err := r.queries.GetCashflowV2(ctx, sqldb.GetCashflowV2Params{
		FromDate: pgtype.Date{Time: from, Valid: true},
		ToDate:   pgtype.Date{Time: to, Valid: true},
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
	report, err := r.GetCashflowV2(ctx, from, to)
	if err != nil {
		return domain.CashflowV2Evidence{}, err
	}

	fromDate := pgtype.Date{Time: from, Valid: true}
	toDate := pgtype.Date{Time: to, Valid: true}

	accounts, err := r.queries.ListAccountsInPeriod(ctx, sqldb.ListAccountsInPeriodParams{
		FromDate: fromDate,
		ToDate:   toDate,
	})
	if err != nil {
		return domain.CashflowV2Evidence{}, fmt.Errorf("list accounts: %w", err)
	}

	transferIDs, err := r.queries.ListConfirmedTransferIDsInPeriod(ctx, sqldb.ListConfirmedTransferIDsInPeriodParams{
		FromDate: fromDate,
		ToDate:   toDate,
	})
	if err != nil {
		return domain.CashflowV2Evidence{}, fmt.Errorf("list transfers: %w", err)
	}

	txIDs, err := r.queries.ListCashflowV2TransactionIDs(ctx, sqldb.ListCashflowV2TransactionIDsParams{
		FromDate: fromDate,
		ToDate:   toDate,
		RowLimit: 50,
	})
	if err != nil {
		return domain.CashflowV2Evidence{}, fmt.Errorf("list evidence txs: %w", err)
	}

	freshness := ""
	latest, err := r.queries.GetLatestBookingDate(ctx)
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
	rows, err := r.queries.GetTopCounterparties(ctx, sqldb.GetTopCounterpartiesParams{
		FromDate: pgtype.Date{Time: from, Valid: true},
		ToDate:   pgtype.Date{Time: to, Valid: true},
		RowLimit: int32(limit),
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
	rows, err := r.queries.ListMonthlyCashflowV2(ctx, sqldb.ListMonthlyCashflowV2Params{
		FromDate: pgtype.Date{Time: from, Valid: true},
		ToDate:   pgtype.Date{Time: to, Valid: true},
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
