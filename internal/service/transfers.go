package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/abteilung6/assetagent/internal/classify"
	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTransferPairNotFound     = errors.New("transfer pair not found")
	ErrTransferPairNotSuggested = errors.New("transfer pair is not suggested")
)

type Transfers struct {
	pool *pgxpool.Pool
}

func NewTransfers(pool *pgxpool.Pool) *Transfers {
	return &Transfers{pool: pool}
}

func (s *Transfers) Scan(ctx context.Context) (domain.TransferScanResult, error) {
	q := sqldb.New(s.pool)

	rows, err := q.ListTransactionsForTransferScan(ctx)
	if err != nil {
		return domain.TransferScanResult{}, fmt.Errorf("list transactions: %w", err)
	}

	legs, err := q.ListTransferPairLegs(ctx)
	if err != nil {
		return domain.TransferScanResult{}, fmt.Errorf("list existing pairs: %w", err)
	}

	existing := make(map[uuid.UUID]struct{}, len(legs)*2)
	for _, leg := range legs {
		existing[leg.TxOutID] = struct{}{}
		existing[leg.TxInID] = struct{}{}
	}

	txs := make([]domain.TransferScanTx, 0, len(rows))
	for _, row := range rows {
		if !row.AccountID.Valid {
			continue
		}
		iban := ""
		if row.CounterpartyIban.Valid {
			iban = row.CounterpartyIban.String
		}
		txs = append(txs, domain.TransferScanTx{
			ID:               row.ID,
			AccountID:        uuid.UUID(row.AccountID.Bytes),
			BookingDate:      row.BookingDate.Time,
			Amount:           row.Amount,
			Purpose:          row.Purpose,
			BookingText:      row.BookingText,
			Counterparty:     row.Counterparty,
			CounterpartyIBAN: iban,
		})
	}

	candidates := classify.DetectTransferCandidates(txs, existing)
	result := domain.TransferScanResult{
		CandidatesConsidered: len(txs),
		SkippedExisting:      len(existing),
	}

	for _, cand := range candidates {
		rationale, err := json.Marshal(cand.Rationale)
		if err != nil {
			return domain.TransferScanResult{}, fmt.Errorf("marshal rationale: %w", err)
		}
		row, err := q.InsertTransferPair(ctx, sqldb.InsertTransferPairParams{
			TxOutID:    cand.TxOutID,
			TxInID:     cand.TxInID,
			Status:     cand.Status,
			Confidence: cand.Confidence,
			Rationale:  rationale,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				continue
			}
			return domain.TransferScanResult{}, fmt.Errorf("insert pair: %w", err)
		}
		result.Suggested++
		result.Pairs = append(result.Pairs, mapTransferPair(row))
	}

	return result, nil
}

func (s *Transfers) List(ctx context.Context) ([]domain.TransferPair, error) {
	rows, err := sqldb.New(s.pool).ListTransferPairs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.TransferPair, len(rows))
	for i, row := range rows {
		out[i] = mapTransferPair(row)
	}
	return out, nil
}

func (s *Transfers) ListCandidates(ctx context.Context) ([]domain.TransferCandidate, error) {
	rows, err := sqldb.New(s.pool).ListSuggestedTransferCandidates(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.TransferCandidate, len(rows))
	for i, row := range rows {
		out[i] = domain.TransferCandidate{
			ID:         row.ID,
			Status:     row.Status,
			Confidence: row.Confidence,
			Amount:     row.OutAmount.Abs(),
			CreatedAt:  row.CreatedAt.Time,
			Out: domain.TransferLegView{
				TransactionID: row.TxOutID,
				AccountName:   row.OutAccountName,
				BookingDate:   row.OutBookingDate.Time,
				Amount:        row.OutAmount,
				BookingText:   row.OutBookingText,
				Purpose:       row.OutPurpose,
				Counterparty:  row.OutCounterparty,
			},
			In: domain.TransferLegView{
				TransactionID: row.TxInID,
				AccountName:   row.InAccountName,
				BookingDate:   row.InBookingDate.Time,
				Amount:        row.InAmount,
				BookingText:   row.InBookingText,
				Purpose:       row.InPurpose,
				Counterparty:  row.InCounterparty,
			},
		}
	}
	return out, nil
}

func (s *Transfers) Confirm(ctx context.Context, id uuid.UUID) (domain.TransferPair, error) {
	return s.decide(ctx, id, true)
}

func (s *Transfers) Reject(ctx context.Context, id uuid.UUID) (domain.TransferPair, error) {
	return s.decide(ctx, id, false)
}

func (s *Transfers) decide(ctx context.Context, id uuid.UUID, confirm bool) (domain.TransferPair, error) {
	q := sqldb.New(s.pool)

	existing, err := q.GetTransferPair(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.TransferPair{}, ErrTransferPairNotFound
		}
		return domain.TransferPair{}, err
	}
	if existing.Status != domain.TransferStatusSuggested {
		return domain.TransferPair{}, ErrTransferPairNotSuggested
	}

	var row sqldb.TransferPair
	if confirm {
		row, err = q.ConfirmTransferPair(ctx, id)
	} else {
		row, err = q.RejectTransferPair(ctx, id)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.TransferPair{}, ErrTransferPairNotSuggested
		}
		return domain.TransferPair{}, err
	}
	return mapTransferPair(row), nil
}

func mapTransferPair(row sqldb.TransferPair) domain.TransferPair {
	var rationale map[string]any
	if len(row.Rationale) > 0 {
		_ = json.Unmarshal(row.Rationale, &rationale)
	}
	pair := domain.TransferPair{
		ID:         row.ID,
		TxOutID:    row.TxOutID,
		TxInID:     row.TxInID,
		Status:     row.Status,
		Confidence: row.Confidence,
		Rationale:  rationale,
		CreatedAt:  row.CreatedAt.Time,
	}
	if row.ConfirmedAt.Valid {
		t := row.ConfirmedAt.Time
		pair.ConfirmedAt = &t
	}
	return pair
}
