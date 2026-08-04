package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestGetRecurring(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	router := newRecurringTestRouter(&stubRecurringService{all: []domain.RecurringSeries{{
		ID:            id,
		DisplayName:   "Netflix",
		Interval:      domain.RecurringIntervalMonthly,
		Kind:          domain.RecurringKindFixed,
		Status:        domain.RecurringStatusActive,
		AmountTypical: decimal.RequireFromString("12.99"),
		AmountLast:    decimal.RequireFromString("12.99"),
		Uncertainty:   domain.RecurringUncertaintyLow,
		MemberCount:   6,
		CreatedAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}}})

	req := httptest.NewRequest(http.MethodGet, "/api/recurring", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{"Netflix", "12.99", "active"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q: %s", want, body)
		}
	}
}

func TestGetUncertainRecurring(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	next := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	router := newRecurringTestRouter(&stubRecurringService{uncertain: []domain.RecurringSeries{{
		ID:            id,
		DisplayName:   "Example Landlord",
		Interval:      domain.RecurringIntervalMonthly,
		Kind:          domain.RecurringKindFixed,
		Status:        domain.RecurringStatusUncertain,
		AmountTypical: decimal.RequireFromString("1200.00"),
		AmountLast:    decimal.RequireFromString("1200.00"),
		NextExpected:  &next,
		Uncertainty:   domain.RecurringUncertaintyLow,
		MemberCount:   3,
		CreatedAt:     time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	}}})

	req := httptest.NewRequest(http.MethodGet, "/api/recurring/uncertain", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{"Example Landlord", "monthly", "1200.00"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q: %s", want, body)
		}
	}
}

func TestPostRecurringConfirm(t *testing.T) {
	id := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	stub := &stubRecurringService{}
	router := newRecurringTestRouter(stub)

	req := httptest.NewRequest(http.MethodPost, "/api/recurring/"+id.String()+"/confirm", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if stub.confirmed != id {
		t.Fatalf("confirmed = %v", stub.confirmed)
	}
}

func TestPostRecurringRejectConflict(t *testing.T) {
	id := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	router := newRecurringTestRouter(&stubRecurringService{
		rejectErr: service.ErrRecurringSeriesNotUncertain,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/recurring/"+id.String()+"/reject", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetRecurringMembers(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	router := newRecurringTestRouter(&stubRecurringService{})
	req := httptest.NewRequest(http.MethodGet, "/api/recurring/"+id.String()+"/members?limit=3", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{"Example Landlord", "Miete Maerz", "-1200.00"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q: %s", want, body)
		}
	}
}

func newRecurringTestRouter(recurring handler.RecurringService) chi.Router {
	router := chi.NewRouter()
	gen.HandlerWithOptions(handler.New(nil, nil, nil, nil, nil, nil, nil, recurring, nil, nil, nil, nil, nil), gen.ChiServerOptions{
		BaseRouter: router,
	})
	return router
}

type stubRecurringService struct {
	all       []domain.RecurringSeries
	uncertain []domain.RecurringSeries
	confirmed uuid.UUID
	rejectErr error
}

func (s *stubRecurringService) List(ctx context.Context) ([]domain.RecurringSeries, error) {
	return s.all, nil
}

func (s *stubRecurringService) ListUncertain(ctx context.Context) ([]domain.RecurringSeries, error) {
	return s.uncertain, nil
}

func (s *stubRecurringService) ListMembers(
	ctx context.Context,
	seriesID uuid.UUID,
	limit int,
) ([]domain.RecurringSeriesMember, error) {
	return []domain.RecurringSeriesMember{{
		TransactionID: uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"),
		BookingDate:   time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Amount:        decimal.RequireFromString("-1200.00"),
		Counterparty:  "Example Landlord",
		Purpose:       "Miete Maerz",
	}}, nil
}

func (s *stubRecurringService) Confirm(ctx context.Context, id uuid.UUID) (domain.RecurringSeries, error) {
	s.confirmed = id
	return domain.RecurringSeries{
		ID:            id,
		DisplayName:   "Example Landlord",
		Interval:      domain.RecurringIntervalMonthly,
		Kind:          domain.RecurringKindFixed,
		Status:        domain.RecurringStatusActive,
		AmountTypical: decimal.RequireFromString("1200.00"),
		AmountLast:    decimal.RequireFromString("1200.00"),
		Uncertainty:   domain.RecurringUncertaintyLow,
		MemberCount:   3,
		CreatedAt:     time.Now().UTC(),
	}, nil
}

func (s *stubRecurringService) Reject(ctx context.Context, id uuid.UUID) (domain.RecurringSeries, error) {
	if s.rejectErr != nil {
		return domain.RecurringSeries{}, s.rejectErr
	}
	return domain.RecurringSeries{
		ID:     id,
		Status: domain.RecurringStatusEnded,
	}, nil
}
