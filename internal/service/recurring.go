package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/abteilung6/assetagent/internal/classify"
	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type Recurring struct {
	pool *pgxpool.Pool
}

func NewRecurring(pool *pgxpool.Pool) *Recurring {
	return &Recurring{pool: pool}
}

func (s *Recurring) Scan(ctx context.Context) (domain.RecurringScanResult, error) {
	q := sqldb.New(s.pool)

	rows, err := q.ListTransactionsForRecurringScan(ctx)
	if err != nil {
		return domain.RecurringScanResult{}, fmt.Errorf("list transactions: %w", err)
	}

	existingIDs, err := q.ListRecurringMemberTransactionIDs(ctx)
	if err != nil {
		return domain.RecurringScanResult{}, fmt.Errorf("list existing members: %w", err)
	}
	existing := make(map[uuid.UUID]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		existing[id] = struct{}{}
	}

	// Also skip legs already in transfer pairs — they are not recurring bills.
	legs, err := q.ListTransferPairLegs(ctx)
	if err != nil {
		return domain.RecurringScanResult{}, fmt.Errorf("list transfer legs: %w", err)
	}
	for _, leg := range legs {
		existing[leg.TxOutID] = struct{}{}
		existing[leg.TxInID] = struct{}{}
	}

	txs := make([]domain.RecurringScanTx, 0, len(rows))
	for _, row := range rows {
		if !row.AccountID.Valid {
			continue
		}
		txs = append(txs, domain.RecurringScanTx{
			ID:           row.ID,
			AccountID:    uuid.UUID(row.AccountID.Bytes),
			BookingDate:  row.BookingDate.Time,
			Amount:       row.Amount,
			Counterparty: row.Counterparty,
			Purpose:      row.Purpose,
			BookingText:  row.BookingText,
		})
	}

	candidates := classify.DetectRecurringSeries(txs, existing, time.Now().UTC())
	result := domain.RecurringScanResult{
		TransactionsConsidered: len(txs),
		SkippedExisting:        len(existing),
	}

	for _, cand := range candidates {
		var next pgtype.Date
		if cand.NextExpected != nil {
			next = pgtype.Date{Time: *cand.NextExpected, Valid: true}
		}
		row, err := q.InsertRecurringSeries(ctx, sqldb.InsertRecurringSeriesParams{
			Fingerprint:   cand.Fingerprint,
			DisplayName:   cand.DisplayName,
			Cadence:       cand.Interval,
			Kind:          cand.Kind,
			Status:        cand.Status,
			AmountTypical: cand.AmountTypical,
			AmountLast:    cand.AmountLast,
			AmountChanged: cand.AmountChanged,
			NextExpected:  next,
			Uncertainty:   cand.Uncertainty,
			MemberCount:   int32(cand.MemberCount),
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return domain.RecurringScanResult{}, fmt.Errorf("insert series: %w", err)
		}

		memberAmounts := memberAmountByID(txs, cand.MemberIDs)
		memberDates := memberDateByID(txs, cand.MemberIDs)
		for _, txID := range cand.MemberIDs {
			amount, ok := memberAmounts[txID]
			if !ok {
				continue
			}
			day, ok := memberDates[txID]
			if !ok {
				continue
			}
			if err := q.InsertRecurringSeriesMember(ctx, sqldb.InsertRecurringSeriesMemberParams{
				SeriesID:      row.ID,
				TransactionID: txID,
				BookingDate:   pgtype.Date{Time: day, Valid: true},
				Amount:        amount,
			}); err != nil {
				return domain.RecurringScanResult{}, fmt.Errorf("insert member: %w", err)
			}
		}

		stored := mapRecurringSeries(row)
		stored.MemberIDs = cand.MemberIDs
		result.Suggested++
		result.Series = append(result.Series, stored)
	}

	return result, nil
}

func (s *Recurring) List(ctx context.Context) ([]domain.RecurringSeries, error) {
	rows, err := sqldb.New(s.pool).ListRecurringSeries(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.RecurringSeries, len(rows))
	for i, row := range rows {
		out[i] = mapRecurringSeries(row)
	}
	return out, nil
}

func (s *Recurring) ListUncertain(ctx context.Context) ([]domain.RecurringSeries, error) {
	// Keep the inbox populated after imports without a separate scan step.
	if _, err := s.Scan(ctx); err != nil {
		return nil, fmt.Errorf("scan before queue: %w", err)
	}
	rows, err := sqldb.New(s.pool).ListUncertainRecurringSeries(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.RecurringSeries, len(rows))
	for i, row := range rows {
		out[i] = mapRecurringSeries(row)
	}
	return out, nil
}

func (s *Recurring) Confirm(ctx context.Context, id uuid.UUID) (domain.RecurringSeries, error) {
	row, err := sqldb.New(s.pool).ConfirmRecurringSeries(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.RecurringSeries{}, s.decideLookupError(ctx, id)
		}
		return domain.RecurringSeries{}, err
	}
	return mapRecurringSeries(row), nil
}

func (s *Recurring) Reject(ctx context.Context, id uuid.UUID) (domain.RecurringSeries, error) {
	row, err := sqldb.New(s.pool).RejectRecurringSeries(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.RecurringSeries{}, s.decideLookupError(ctx, id)
		}
		return domain.RecurringSeries{}, err
	}
	return mapRecurringSeries(row), nil
}

func (s *Recurring) decideLookupError(ctx context.Context, id uuid.UUID) error {
	_, err := sqldb.New(s.pool).GetRecurringSeries(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRecurringSeriesNotFound
	}
	if err != nil {
		return err
	}
	return ErrRecurringSeriesNotUncertain
}

var (
	ErrRecurringSeriesNotFound     = errors.New("recurring series not found")
	ErrRecurringSeriesNotUncertain = errors.New("recurring series is not uncertain")
)

func mapRecurringSeries(row sqldb.RecurringSeries) domain.RecurringSeries {
	out := domain.RecurringSeries{
		ID:            row.ID,
		Fingerprint:   row.Fingerprint,
		DisplayName:   row.DisplayName,
		Interval:      row.Cadence,
		Kind:          row.Kind,
		Status:        row.Status,
		AmountTypical: row.AmountTypical,
		AmountLast:    row.AmountLast,
		AmountChanged: row.AmountChanged,
		Uncertainty:   row.Uncertainty,
		MemberCount:   int(row.MemberCount),
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}
	if row.NextExpected.Valid {
		t := row.NextExpected.Time
		out.NextExpected = &t
	}
	return out
}

func memberAmountByID(txs []domain.RecurringScanTx, ids []uuid.UUID) map[uuid.UUID]decimal.Decimal {
	wanted := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		wanted[id] = struct{}{}
	}
	out := make(map[uuid.UUID]decimal.Decimal, len(ids))
	for _, tx := range txs {
		if _, ok := wanted[tx.ID]; ok {
			out[tx.ID] = tx.Amount
		}
	}
	return out
}

func memberDateByID(txs []domain.RecurringScanTx, ids []uuid.UUID) map[uuid.UUID]time.Time {
	wanted := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		wanted[id] = struct{}{}
	}
	out := make(map[uuid.UUID]time.Time, len(ids))
	for _, tx := range txs {
		if _, ok := wanted[tx.ID]; ok {
			out[tx.ID] = tx.BookingDate
		}
	}
	return out
}
