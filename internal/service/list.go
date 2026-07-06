package service

import (
	"context"
	"fmt"

	"github.com/abteilung6/assetagent/internal/domain"
)

type ListRepository interface {
	List(ctx context.Context, params domain.ListParams) (domain.ListResult, error)
}

type List struct {
	repo ListRepository
}

func NewList(repo ListRepository) *List {
	return &List{repo: repo}
}

type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

func (s *List) ListTransactions(ctx context.Context, params domain.ListParams) (domain.ListResult, error) {
	if params.Limit <= 0 {
		params.Limit = domain.DefaultListLimit
	}
	if params.Limit > domain.MaxListLimit {
		return domain.ListResult{}, ValidationError{Message: fmt.Sprintf("limit must be between 1 and %d", domain.MaxListLimit)}
	}
	if params.Offset < 0 {
		return domain.ListResult{}, ValidationError{Message: "offset must be non-negative"}
	}
	if params.Sort != "" && !isValidSortField(params.Sort) {
		return domain.ListResult{}, ValidationError{Message: fmt.Sprintf("sort must be one of: booking_date, amount, counterparty")}
	}
	if params.MinAmount != nil && params.MaxAmount != nil && params.MinAmount.GreaterThan(*params.MaxAmount) {
		return domain.ListResult{}, ValidationError{Message: "min_amount must be less than or equal to max_amount"}
	}

	return s.repo.List(ctx, params)
}

func isValidSortField(field domain.SortField) bool {
	switch field {
	case domain.SortBookingDate, domain.SortAmount, domain.SortCounterparty:
		return true
	default:
		return false
	}
}
