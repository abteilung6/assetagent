package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/shopspring/decimal"
)

type BaselineService interface {
	RecomputeAndSave(ctx context.Context, from, to *time.Time) (service.ComputedBaseline, error)
	Current(ctx context.Context) (service.ComputedBaseline, error)
	Confirm(ctx context.Context, id uuid.UUID) (service.ComputedBaseline, error)
	Adjust(ctx context.Context, id uuid.UUID, metricKey string, newValue decimal.Decimal, reason string) (service.ComputedBaseline, error)
	MonthlyCashflow(ctx context.Context, months int) ([]service.MonthlyCashflowPoint, error)
	OneOffImpact(ctx context.Context, from, to time.Time) (repository.OneOffExpenseImpact, error)
	CategorySpend(ctx context.Context, from, to time.Time, limit int) ([]repository.CategorySpendPoint, error)
	CategoryMerchants(ctx context.Context, from, to time.Time, categorySlug string, limit int) ([]repository.CategoryMerchantSpendPoint, error)
	DailyExpensePace(ctx context.Context, from, to time.Time) ([]repository.DailyExpensePacePoint, error)
}

func (h *Handler) GetCurrentBaseline(w http.ResponseWriter, r *http.Request) {
	if h.baseline == nil {
		writeInternalError(w, "baseline service is not configured")
		return
	}
	baseline, err := h.baseline.Current(r.Context())
	if err != nil {
		if errors.Is(err, service.ErrBaselineNone) {
			writeNotFoundError(w, "no baseline available")
			return
		}
		writeInternalError(w, "failed to load current baseline")
		return
	}
	writeJSON(w, http.StatusOK, toAPIBaseline(baseline))
}

func (h *Handler) GetBaselineMonthlyCashflow(w http.ResponseWriter, r *http.Request, params gen.GetBaselineMonthlyCashflowParams) {
	if h.baseline == nil {
		writeInternalError(w, "baseline service is not configured")
		return
	}
	months := 6
	if params.Months != nil {
		months = *params.Months
	}
	items, err := h.baseline.MonthlyCashflow(r.Context(), months)
	if err != nil {
		writeInternalError(w, "failed to load monthly cashflow")
		return
	}
	data := make([]gen.BaselineMonthlyCashflowPoint, len(items))
	for i, item := range items {
		data[i] = gen.BaselineMonthlyCashflowPoint{
			MonthStart: openapi_types.Date{Time: item.MonthStart},
			Income:     item.Income.StringFixed(2),
			Expenses:   item.Expenses.StringFixed(2),
			Net:        item.Net.StringFixed(2),
		}
	}
	writeJSON(w, http.StatusOK, gen.BaselineMonthlyCashflowResponse{Data: data})
}

func (h *Handler) GetBaselineOneOffImpact(w http.ResponseWriter, r *http.Request, params gen.GetBaselineOneOffImpactParams) {
	if h.baseline == nil {
		writeInternalError(w, "baseline service is not configured")
		return
	}
	impact, err := h.baseline.OneOffImpact(r.Context(), params.From.Time, params.To.Time)
	if err != nil {
		if errors.Is(err, service.ErrInvalidBaselinePeriod) {
			writeValidationError(w, err.Error())
			return
		}
		writeInternalError(w, "failed to load one-off impact")
		return
	}
	writeJSON(w, http.StatusOK, gen.BaselineOneOffImpact{
		Count:        impact.Count,
		ExpenseTotal: impact.ExpenseTotal.StringFixed(2),
	})
}

func (h *Handler) GetBaselineCategorySpend(w http.ResponseWriter, r *http.Request, params gen.GetBaselineCategorySpendParams) {
	if h.baseline == nil {
		writeInternalError(w, "baseline service is not configured")
		return
	}
	limit := 8
	if params.Limit != nil {
		limit = *params.Limit
	}
	items, err := h.baseline.CategorySpend(r.Context(), params.From.Time, params.To.Time, limit)
	if err != nil {
		if errors.Is(err, service.ErrInvalidBaselinePeriod) {
			writeValidationError(w, err.Error())
			return
		}
		writeInternalError(w, "failed to load category spend")
		return
	}
	data := make([]gen.BaselineCategorySpendPoint, len(items))
	for i, item := range items {
		data[i] = gen.BaselineCategorySpendPoint{
			CategorySlug:     item.CategorySlug,
			CategoryName:     item.CategoryName,
			Total:            item.Total.StringFixed(2),
			TransactionCount: item.TransactionCount,
		}
	}
	writeJSON(w, http.StatusOK, gen.BaselineCategorySpendResponse{Data: data})
}

func (h *Handler) GetBaselineCategoryMerchants(w http.ResponseWriter, r *http.Request, params gen.GetBaselineCategoryMerchantsParams) {
	if h.baseline == nil {
		writeInternalError(w, "baseline service is not configured")
		return
	}
	limit := 8
	if params.Limit != nil {
		limit = *params.Limit
	}
	items, err := h.baseline.CategoryMerchants(
		r.Context(),
		params.From.Time,
		params.To.Time,
		params.CategorySlug,
		limit,
	)
	if err != nil {
		if errors.Is(err, service.ErrInvalidBaselinePeriod) {
			writeValidationError(w, err.Error())
			return
		}
		writeInternalError(w, "failed to load category merchants")
		return
	}
	data := make([]gen.BaselineCategoryMerchantPoint, len(items))
	for i, item := range items {
		data[i] = gen.BaselineCategoryMerchantPoint{
			Merchant:         item.Merchant,
			Total:            item.Total.StringFixed(2),
			TransactionCount: item.TransactionCount,
		}
	}
	writeJSON(w, http.StatusOK, gen.BaselineCategoryMerchantsResponse{Data: data})
}

func (h *Handler) GetBaselineDailyExpensePace(w http.ResponseWriter, r *http.Request, params gen.GetBaselineDailyExpensePaceParams) {
	if h.baseline == nil {
		writeInternalError(w, "baseline service is not configured")
		return
	}
	items, err := h.baseline.DailyExpensePace(r.Context(), params.From.Time, params.To.Time)
	if err != nil {
		if errors.Is(err, service.ErrInvalidBaselinePeriod) {
			writeValidationError(w, err.Error())
			return
		}
		writeInternalError(w, "failed to load daily expense pace")
		return
	}
	data := make([]gen.BaselineDailyExpensePacePoint, len(items))
	for i, item := range items {
		data[i] = gen.BaselineDailyExpensePacePoint{
			Date:             openapi_types.Date{Time: item.Date},
			Expenses:         item.Expenses.StringFixed(2),
			TransactionCount: item.TransactionCount,
		}
	}
	writeJSON(w, http.StatusOK, gen.BaselineDailyExpensePaceResponse{Data: data})
}

func (h *Handler) PostBaselinesRecompute(w http.ResponseWriter, r *http.Request) {
	if h.baseline == nil {
		writeInternalError(w, "baseline service is not configured")
		return
	}

	var body gen.BaselineRecomputeRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeValidationError(w, "invalid JSON body")
			return
		}
	}

	var from, to *time.Time
	if body.From != nil || body.To != nil {
		if body.From == nil || body.To == nil {
			writeValidationError(w, "from and to must both be set")
			return
		}
		f := body.From.Time
		t := body.To.Time
		from, to = &f, &t
	}

	baseline, err := h.baseline.RecomputeAndSave(r.Context(), from, to)
	if err != nil {
		if errors.Is(err, service.ErrInvalidBaselinePeriod) {
			writeValidationError(w, err.Error())
			return
		}
		writeInternalError(w, "failed to recompute baseline")
		return
	}
	writeJSON(w, http.StatusOK, toAPIBaseline(baseline))
}

func (h *Handler) PostBaselineConfirm(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	if h.baseline == nil {
		writeInternalError(w, "baseline service is not configured")
		return
	}
	baseline, err := h.baseline.Confirm(r.Context(), id)
	if err != nil {
		writeBaselineDecideError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIBaseline(baseline))
}

func (h *Handler) PostBaselineAdjust(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	if h.baseline == nil {
		writeInternalError(w, "baseline service is not configured")
		return
	}

	var body gen.BaselineAdjustRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	newValue, err := decimal.NewFromString(body.NewValue)
	if err != nil {
		writeValidationError(w, "invalid new_value")
		return
	}

	baseline, err := h.baseline.Adjust(r.Context(), id, string(body.MetricKey), newValue, body.Reason)
	if err != nil {
		writeBaselineAdjustError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIBaseline(baseline))
}

func writeBaselineDecideError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrBaselineNotFound):
		writeNotFoundError(w, err.Error())
	case errors.Is(err, service.ErrBaselineNotDraft):
		writeConflictError(w, err.Error())
	default:
		writeInternalError(w, "failed to confirm baseline")
	}
}

func writeBaselineAdjustError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrBaselineNotFound):
		writeNotFoundError(w, err.Error())
	case errors.Is(err, service.ErrBaselineNotAdjustable):
		writeConflictError(w, err.Error())
	case errors.Is(err, service.ErrInvalidBaselineMetric),
		errors.Is(err, service.ErrEmptyAdjustReason),
		errors.Is(err, service.ErrInvalidBaselineAmount):
		writeValidationError(w, err.Error())
	default:
		writeInternalError(w, "failed to adjust baseline")
	}
}

func toAPIBaseline(b service.ComputedBaseline) gen.FinancialBaseline {
	metrics := make([]gen.BaselineMetric, len(b.Metrics))
	for i, m := range b.Metrics {
		ids := m.EvidenceIDs
		if ids == nil {
			ids = []string{}
		}
		metric := gen.BaselineMetric{
			Key:         gen.BaselineMetricKey(m.Key),
			Value:       m.Value.StringFixed(2),
			Calculation: m.Calculation,
			Confidence:  gen.BaselineMetricConfidence(m.Confidence),
			EvidenceIds: ids,
		}
		if len(m.Assumptions) > 0 {
			assumptions := m.Assumptions
			metric.Assumptions = &assumptions
		}
		metrics[i] = metric
	}
	assumptions := b.Assumptions
	if assumptions == nil {
		assumptions = []string{}
	}
	out := gen.FinancialBaseline{
		Id:                      b.ID,
		PeriodFrom:              openapi_types.Date{Time: dateOnly(b.PeriodFrom)},
		PeriodTo:                openapi_types.Date{Time: dateOnly(b.PeriodTo)},
		AlgorithmVersion:        b.AlgorithmVersion,
		Status:                  gen.FinancialBaselineStatus(b.Status),
		RegularMonthlyIncome:    b.RegularMonthlyIncome.StringFixed(2),
		MonthlyFixedCosts:       b.MonthlyFixedCosts.StringFixed(2),
		MonthlyIrregularCosts:   b.MonthlyIrregularCosts.StringFixed(2),
		AvgVariableSpend:        b.AvgVariableSpend.StringFixed(2),
		SustainableFreeCashflow: b.SustainableFreeCashflow.StringFixed(2),
		Confidence:              gen.FinancialBaselineConfidence(b.Confidence),
		Assumptions:             assumptions,
		Metrics:                 metrics,
		CreatedAt:               b.CreatedAt.UTC(),
	}
	if b.ConfirmedAt != nil {
		t := b.ConfirmedAt.UTC()
		out.ConfirmedAt = &t
	}
	return out
}
