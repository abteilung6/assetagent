package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/finance"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestGetCurrentBaseline(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	router := newBaselineTestRouter(&stubBaselineService{current: sampleBaseline(id)})

	req := httptest.NewRequest(http.MethodGet, "/api/baselines/current", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{id.String(), "3500.00", "1800.00", "draft"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q: %s", want, body)
		}
	}
}

func TestGetCurrentBaselineNone(t *testing.T) {
	router := newBaselineTestRouter(&stubBaselineService{currentErr: service.ErrBaselineNone})
	req := httptest.NewRequest(http.MethodGet, "/api/baselines/current", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestPostBaselinesRecompute(t *testing.T) {
	id := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	stub := &stubBaselineService{saved: sampleBaseline(id)}
	router := newBaselineTestRouter(stub)

	req := httptest.NewRequest(http.MethodPost, "/api/baselines/recompute", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !stub.recomputed {
		t.Fatal("expected recompute")
	}
}

func TestPostBaselineConfirm(t *testing.T) {
	id := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	stub := &stubBaselineService{}
	router := newBaselineTestRouter(stub)

	req := httptest.NewRequest(http.MethodPost, "/api/baselines/"+id.String()+"/confirm", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if stub.confirmed != id {
		t.Fatalf("confirmed = %v", stub.confirmed)
	}
}

func TestPostBaselineConfirmConflict(t *testing.T) {
	id := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	router := newBaselineTestRouter(&stubBaselineService{
		confirmErr: service.ErrBaselineNotDraft,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/baselines/"+id.String()+"/confirm", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestPostBaselineAdjust(t *testing.T) {
	id := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	newID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")
	stub := &stubBaselineService{adjusted: sampleBaseline(newID)}
	router := newBaselineTestRouter(stub)

	body := `{"metric_key":"monthly_fixed_costs","new_value":"1100.00","reason":"Rent reduced"}`
	req := httptest.NewRequest(http.MethodPost, "/api/baselines/"+id.String()+"/adjust", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if stub.adjustID != id {
		t.Fatalf("adjustID = %v", stub.adjustID)
	}
	if stub.adjustKey != finance.MetricMonthlyFixedCosts {
		t.Fatalf("key = %q", stub.adjustKey)
	}
	if !stub.adjustValue.Equal(decimal.RequireFromString("1100.00")) {
		t.Fatalf("value = %s", stub.adjustValue)
	}
}

func TestPostBaselineAdjustValidation(t *testing.T) {
	id := uuid.MustParse("12121212-1212-1212-1212-121212121212")
	router := newBaselineTestRouter(&stubBaselineService{})
	body := `{"metric_key":"monthly_fixed_costs","new_value":"not-a-number","reason":"x"}`
	req := httptest.NewRequest(http.MethodPost, "/api/baselines/"+id.String()+"/adjust", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func newBaselineTestRouter(baseline handler.BaselineService) chi.Router {
	router := chi.NewRouter()
	gen.HandlerWithOptions(handler.New(nil, nil, nil, nil, nil, nil, nil, nil, baseline, nil, nil, nil), gen.ChiServerOptions{
		BaseRouter: router,
	})
	return router
}

func sampleBaseline(id uuid.UUID) service.ComputedBaseline {
	return service.ComputedBaseline{
		ID:                      id,
		PeriodFrom:              time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		PeriodTo:                time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
		AlgorithmVersion:        finance.AlgorithmVersion,
		Status:                  service.BaselineStatusDraft,
		RegularMonthlyIncome:    decimal.RequireFromString("3500.00"),
		MonthlyFixedCosts:       decimal.RequireFromString("1200.00"),
		MonthlyIrregularCosts:   decimal.RequireFromString("50.00"),
		AvgVariableSpend:        decimal.RequireFromString("450.00"),
		SustainableFreeCashflow: decimal.RequireFromString("1800.00"),
		Confidence:              finance.ConfidenceHigh,
		Assumptions:             []string{"period=explicit"},
		Metrics: []finance.MetricEvidence{{
			Key: finance.MetricSustainableFreeCash, Value: decimal.RequireFromString("1800.00"),
			Calculation: "derived", Confidence: finance.ConfidenceHigh,
		}},
		CreatedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
	}
}

type stubBaselineService struct {
	current     service.ComputedBaseline
	currentErr  error
	saved       service.ComputedBaseline
	recomputed  bool
	confirmed   uuid.UUID
	confirmErr  error
	adjusted    service.ComputedBaseline
	adjustID    uuid.UUID
	adjustKey   string
	adjustValue decimal.Decimal
	adjustErr   error
}

func (s *stubBaselineService) RecomputeAndSave(ctx context.Context, from, to *time.Time) (service.ComputedBaseline, error) {
	s.recomputed = true
	return s.saved, nil
}

func (s *stubBaselineService) Current(ctx context.Context) (service.ComputedBaseline, error) {
	if s.currentErr != nil {
		return service.ComputedBaseline{}, s.currentErr
	}
	return s.current, nil
}

func (s *stubBaselineService) Confirm(ctx context.Context, id uuid.UUID) (service.ComputedBaseline, error) {
	if s.confirmErr != nil {
		return service.ComputedBaseline{}, s.confirmErr
	}
	s.confirmed = id
	out := sampleBaseline(id)
	out.Status = service.BaselineStatusConfirmed
	return out, nil
}

func (s *stubBaselineService) Adjust(
	ctx context.Context,
	id uuid.UUID,
	metricKey string,
	newValue decimal.Decimal,
	reason string,
) (service.ComputedBaseline, error) {
	if s.adjustErr != nil {
		return service.ComputedBaseline{}, s.adjustErr
	}
	s.adjustID = id
	s.adjustKey = metricKey
	s.adjustValue = newValue
	return s.adjusted, nil
}

// Ensure JSON round-trip for sample payloads in tests.
var _ = json.Marshal
