package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/abteilung6/assetagent/internal/classify"
	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
			if err == pgx.ErrNoRows {
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
