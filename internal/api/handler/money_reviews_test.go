package handler_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/review"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestGetMoneyReviews(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	router := newMoneyReviewTestRouter(&stubMoneyReviewService{
		list: []service.MoneyReview{sampleMoneyReview(id)},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/reviews", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Money review for") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestPostMoneyReviewsRequiresBaseline(t *testing.T) {
	router := newMoneyReviewTestRouter(&stubMoneyReviewService{
		createErr: service.ErrBaselineRequired,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/reviews", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestPostMoneyReviewConfirm(t *testing.T) {
	id := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	stub := &stubMoneyReviewService{}
	router := newMoneyReviewTestRouter(stub)
	req := httptest.NewRequest(http.MethodPost, "/api/reviews/"+id.String()+"/confirm", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if stub.confirmed != id {
		t.Fatalf("confirmed = %v", stub.confirmed)
	}
}

func TestGetMoneyReviewNotFound(t *testing.T) {
	id := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	router := newMoneyReviewTestRouter(&stubMoneyReviewService{
		getErr: service.ErrMoneyReviewNotFound,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/reviews/"+id.String(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func newMoneyReviewTestRouter(svc handler.MoneyReviewService) chi.Router {
	router := chi.NewRouter()
	gen.HandlerWithOptions(handler.New(nil, nil, nil, nil, nil, nil, nil, nil, nil, svc, nil, nil, nil), gen.ChiServerOptions{
		BaseRouter: router,
	})
	return router
}

func sampleMoneyReview(id uuid.UUID) service.MoneyReview {
	amount := decimal.RequireFromString("-200.00")
	return service.MoneyReview{
		ID:         id,
		BaselineID: uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd"),
		PeriodFrom: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		PeriodTo:   time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
		Status:     service.MoneyReviewStatusNeedsConfirmation,
		Summary:    "Money review for 2026-03-01 – 2026-03-31: 1 finding(s). Free cashflow -200.00 €/month.",
		Findings: []review.Finding{{
			Type:        review.FindingFreeCashflowPressure,
			Title:       "Sustainable free cashflow is negative",
			Amount:      &amount,
			PeriodFrom:  time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			PeriodTo:    time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
			Confidence:  review.ConfidenceHigh,
			EvidenceIDs: []string{"baseline_free_cashflow"},
		}},
		CreatedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	}
}

type stubMoneyReviewService struct {
	list      []service.MoneyReview
	createErr error
	getErr    error
	confirmed uuid.UUID
}

func (s *stubMoneyReviewService) Create(ctx context.Context, baselineID *uuid.UUID) (service.MoneyReview, error) {
	if s.createErr != nil {
		return service.MoneyReview{}, s.createErr
	}
	return sampleMoneyReview(uuid.New()), nil
}

func (s *stubMoneyReviewService) Get(ctx context.Context, id uuid.UUID) (service.MoneyReview, error) {
	if s.getErr != nil {
		return service.MoneyReview{}, s.getErr
	}
	return sampleMoneyReview(id), nil
}

func (s *stubMoneyReviewService) List(ctx context.Context, limit int) ([]service.MoneyReview, error) {
	return s.list, nil
}

func (s *stubMoneyReviewService) Confirm(ctx context.Context, id uuid.UUID) (service.MoneyReview, error) {
	s.confirmed = id
	out := sampleMoneyReview(id)
	out.Status = service.MoneyReviewStatusConfirmed
	return out, nil
}
