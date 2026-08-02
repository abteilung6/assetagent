package repository

import (
	"context"
	"time"

	"github.com/abteilung6/assetagent/internal/domain"
	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
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
