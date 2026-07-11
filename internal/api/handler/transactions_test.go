package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type stubListService struct {
	result domain.ListResult
	err    error
	params domain.ListParams
}

func (s *stubListService) ListTransactions(ctx context.Context, params domain.ListParams) (domain.ListResult, error) {
	s.params = params
	return s.result, s.err
}

func TestGetTransactions_returnsPaginatedList(t *testing.T) {
	txID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	list := &stubListService{
		result: domain.ListResult{
			Transactions: []domain.Transaction{
				{
					ID:            txID,
					OrderAccount:  "DE15100500006011880043",
					BookingDate:   time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC),
					ValueDate:     time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC),
					BookingText:   "FOLGELASTSCHRIFT",
					Purpose:       "Prime Video",
					Counterparty:  "AMAZON",
					Amount:        decimal.RequireFromString("-2.99"),
					Currency:      "EUR",
					Info:          "Umsatz gebucht",
				},
			},
			Total: 42,
		},
	}

	router := chi.NewRouter()
	gen.HandlerWithOptions(handler.New(list, nil, nil), gen.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handler.APIErrorHandler,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/transactions?limit=10&offset=5", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp gen.TransactionListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Pagination.Limit != 10 || resp.Pagination.Offset != 5 || resp.Pagination.Total != 42 {
		t.Fatalf("pagination = %+v, want limit=10 offset=5 total=42", resp.Pagination)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("len(data) = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].Id.String() != txID.String() {
		t.Fatalf("id = %s, want %s", resp.Data[0].Id, txID)
	}
	if resp.Data[0].Amount != "-2.99" {
		t.Fatalf("amount = %q, want -2.99", resp.Data[0].Amount)
	}
	if list.params.Limit != 10 || list.params.Offset != 5 {
		t.Fatalf("service params = %+v, want limit=10 offset=5", list.params)
	}
}

func TestGetTransactions_invalidDate_returns400(t *testing.T) {
	router := chi.NewRouter()
	gen.HandlerWithOptions(handler.New(&stubListService{}, nil, nil), gen.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handler.APIErrorHandler,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/transactions?from=not-a-date", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	var resp gen.Error
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error != "validation_failed" {
		t.Fatalf("error = %q, want validation_failed", resp.Error)
	}
}

func TestGetTransactions_mapsFilterParams(t *testing.T) {
	list := &stubListService{result: domain.ListResult{Total: 0}}

	router := chi.NewRouter()
	gen.HandlerWithOptions(handler.New(list, nil, nil), gen.ChiServerOptions{
		BaseRouter:       router,
		ErrorHandlerFunc: handler.APIErrorHandler,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/transactions?account=DE15100500006011880043&counterparty=AMAZON&min_amount=-10&max_amount=0&sort=amount&order=asc&q=Prime", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if list.params.Account == nil || *list.params.Account != "DE15100500006011880043" {
		t.Fatalf("account = %v, want DE15100500006011880043", list.params.Account)
	}
	if list.params.Counterparty == nil || *list.params.Counterparty != "AMAZON" {
		t.Fatalf("counterparty = %v, want AMAZON", list.params.Counterparty)
	}
	if list.params.Search == nil || *list.params.Search != "Prime" {
		t.Fatalf("search = %v, want Prime", list.params.Search)
	}
	if list.params.Sort != domain.SortAmount || !list.params.SortAsc {
		t.Fatalf("sort = %q asc=%v, want amount true", list.params.Sort, list.params.SortAsc)
	}
	if list.params.MinAmount == nil || !list.params.MinAmount.Equal(decimal.RequireFromString("-10")) {
		t.Fatalf("min_amount = %v", list.params.MinAmount)
	}
	if list.params.MaxAmount == nil || !list.params.MaxAmount.Equal(decimal.RequireFromString("0")) {
		t.Fatalf("max_amount = %v", list.params.MaxAmount)
	}
}
