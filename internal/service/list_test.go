package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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

func (f *fakeListRepo) SetOneOff(ctx context.Context, id uuid.UUID, oneOff bool) (domain.Transaction, error) {
	return domain.Transaction{ID: id, OneOff: oneOff}, f.err
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
	if !errors.Is(err, service.ErrInvalidLimit) {
		t.Fatalf("error = %v, want ErrInvalidLimit", err)
	}
}

func TestListTransactions_rejectsNegativeOffset(t *testing.T) {
	svc := service.NewList(&fakeListRepo{})

	_, err := svc.ListTransactions(context.Background(), domain.ListParams{Offset: -1})
	if err == nil {
		t.Fatal("expected error for negative offset")
	}
	if !errors.Is(err, service.ErrInvalidOffset) {
		t.Fatalf("error = %v, want ErrInvalidOffset", err)
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

func TestListTransactions_rejectsInvalidSort(t *testing.T) {
	svc := service.NewList(&fakeListRepo{})

	_, err := svc.ListTransactions(context.Background(), domain.ListParams{
		Sort: domain.SortField("unknown"),
	})
	if err == nil {
		t.Fatal("expected error for invalid sort")
	}
	if !errors.Is(err, service.ErrInvalidSort) {
		t.Fatalf("error = %v, want ErrInvalidSort", err)
	}
}

func TestListTransactions_rejectsMinGreaterThanMax(t *testing.T) {
	svc := service.NewList(&fakeListRepo{})

	min := decimal.RequireFromString("100")
	max := decimal.RequireFromString("10")

	_, err := svc.ListTransactions(context.Background(), domain.ListParams{
		MinAmount: &min,
		MaxAmount: &max,
	})
	if err == nil {
		t.Fatal("expected error for min > max")
	}
	if !errors.Is(err, service.ErrInvalidAmountRange) {
		t.Fatalf("error = %v, want ErrInvalidAmountRange", err)
	}
}

func TestListTransactions_passesFilters(t *testing.T) {
	repo := &fakeListRepo{result: domain.ListResult{Total: 0}}
	svc := service.NewList(repo)

	account := "DE15100500006011880043"
	counterparty := "AMAZON"
	search := "Prime"
	min := decimal.RequireFromString("-10")
	max := decimal.RequireFromString("0")

	_, err := svc.ListTransactions(context.Background(), domain.ListParams{
		Account:      &account,
		Counterparty: &counterparty,
		Search:       &search,
		MinAmount:    &min,
		MaxAmount:    &max,
		Sort:         domain.SortAmount,
		SortAsc:      true,
	})
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	if repo.params.Account == nil || *repo.params.Account != account {
		t.Fatalf("account not passed through")
	}
	if repo.params.Counterparty == nil || *repo.params.Counterparty != counterparty {
		t.Fatalf("counterparty not passed through")
	}
	if repo.params.Search == nil || *repo.params.Search != search {
		t.Fatalf("search not passed through")
	}
	if !repo.params.SortAsc || repo.params.Sort != domain.SortAmount {
		t.Fatalf("sort params = %q asc=%v, want amount true", repo.params.Sort, repo.params.SortAsc)
	}
}
