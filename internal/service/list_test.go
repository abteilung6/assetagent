package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/service"
)

type fakeListRepo struct {
	params domain.ListParams
	result domain.ListResult
	err    error
}

func (f *fakeListRepo) List(ctx context.Context, params domain.ListParams) (domain.ListResult, error) {
	f.params = params
	return f.result, f.err
}

func TestListTransactions_defaultsLimit(t *testing.T) {
	repo := &fakeListRepo{result: domain.ListResult{Total: 0}}
	svc := service.NewList(repo)

	_, err := svc.ListTransactions(context.Background(), domain.ListParams{})
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	if repo.params.Limit != domain.DefaultListLimit {
		t.Fatalf("limit = %d, want %d", repo.params.Limit, domain.DefaultListLimit)
	}
}

func TestListTransactions_rejectsLimitAboveMax(t *testing.T) {
	svc := service.NewList(&fakeListRepo{})

	_, err := svc.ListTransactions(context.Background(), domain.ListParams{Limit: 201})
	if err == nil {
		t.Fatal("expected error for limit > max")
	}
	var validationErr service.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error type = %T, want ValidationError", err)
	}
}

func TestListTransactions_rejectsNegativeOffset(t *testing.T) {
	svc := service.NewList(&fakeListRepo{})

	_, err := svc.ListTransactions(context.Background(), domain.ListParams{Offset: -1})
	if err == nil {
		t.Fatal("expected error for negative offset")
	}
	var validationErr service.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("error type = %T, want ValidationError", err)
	}
}

func TestListTransactions_passesDateFilters(t *testing.T) {
	repo := &fakeListRepo{result: domain.ListResult{Total: 1}}
	svc := service.NewList(repo)

	from := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	_, err := svc.ListTransactions(context.Background(), domain.ListParams{
		Limit:    10,
		Offset:   5,
		FromDate: &from,
		ToDate:   &to,
	})
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	if repo.params.Limit != 10 || repo.params.Offset != 5 {
		t.Fatalf("limit/offset = %d/%d, want 10/5", repo.params.Limit, repo.params.Offset)
	}
	if repo.params.FromDate == nil || !repo.params.FromDate.Equal(from) {
		t.Fatalf("from date not passed through")
	}
	if repo.params.ToDate == nil || !repo.params.ToDate.Equal(to) {
		t.Fatalf("to date not passed through")
	}
}
