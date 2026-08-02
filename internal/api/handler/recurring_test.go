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

func newRecurringTestRouter(recurring handler.RecurringService) chi.Router {
	router := chi.NewRouter()
	gen.HandlerWithOptions(handler.New(nil, nil, nil, nil, nil, nil, nil, recurring), gen.ChiServerOptions{
		BaseRouter: router,
	})
	return router
}

type stubRecurringService struct {
	uncertain []domain.RecurringSeries
	confirmed uuid.UUID
	rejectErr error
}

func (s *stubRecurringService) ListUncertain(ctx context.Context) ([]domain.RecurringSeries, error) {
	return s.uncertain, nil
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
