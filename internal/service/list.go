package service

import (
	"context"
	"errors"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ListRepository interface {
	List(ctx context.Context, params domain.ListParams) (domain.ListResult, error)
	SetOneOff(ctx context.Context, id uuid.UUID, oneOff bool) (domain.Transaction, error)
}

type List struct {
	repo ListRepository
}

func NewList(repo ListRepository) *List {
	return &List{repo: repo}
}

func (s *List) ListTransactions(ctx context.Context, params domain.ListParams) (domain.ListResult, error) {
	if params.Limit <= 0 {
		params.Limit = domain.DefaultListLimit
	}
	if params.Limit > domain.MaxListLimit {
		return domain.ListResult{}, ErrInvalidLimit
	}
	if params.Offset < 0 {
		return domain.ListResult{}, ErrInvalidOffset
	}
	if params.Sort != "" && !isValidSortField(params.Sort) {
		return domain.ListResult{}, ErrInvalidSort
	}
	if params.MinAmount != nil && params.MaxAmount != nil && params.MinAmount.GreaterThan(*params.MaxAmount) {
		return domain.ListResult{}, ErrInvalidAmountRange
	}

	return s.repo.List(ctx, params)
}

func (s *List) SetTransactionOneOff(ctx context.Context, id uuid.UUID, oneOff bool) (domain.Transaction, error) {
	tx, err := s.repo.SetOneOff(ctx, id, oneOff)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Transaction{}, ErrTransactionNotFound
	}
	if err != nil {
		return domain.Transaction{}, err
	}
	return tx, nil
}

func isValidSortField(field domain.SortField) bool {
	switch field {
	case domain.SortBookingDate, domain.SortAmount, domain.SortCounterparty:
		return true
	default:
		return false
	}
}
