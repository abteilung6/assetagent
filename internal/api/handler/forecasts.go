package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/forecast"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/shopspring/decimal"
)

type ForecastService interface {
	Create(ctx context.Context, req service.CreateForecastRequest) (service.ForecastArtifact, error)
	Get(ctx context.Context, id uuid.UUID) (service.ForecastArtifact, error)
	LatestForCurrentBaseline(ctx context.Context) (service.ForecastArtifact, error)
	RunScenario(ctx context.Context, req service.RunScenarioRequest) (service.ScenarioArtifact, error)
	ListScenarios(ctx context.Context, forecastID uuid.UUID) ([]service.ScenarioArtifact, error)
}

func (h *Handler) PostForecasts(w http.ResponseWriter, r *http.Request) {
	if h.forecast == nil {
		writeInternalError(w, "forecast service is not configured")
		return
	}
	var body gen.ForecastCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	startBal, err := decimal.NewFromString(body.StartingBalance)
	if err != nil {
		writeValidationError(w, "invalid starting_balance")
		return
	}
	req := service.CreateForecastRequest{StartingBalance: startBal}
	if body.BaselineId != nil {
		id := uuid.UUID(*body.BaselineId)
		req.BaselineID = &id
	}
	if body.HorizonDays != nil {
		req.HorizonDays = *body.HorizonDays
	}
	if body.Assumptions != nil {
		a := forecast.Assumptions{
			DisabledSeriesIDs: body.Assumptions.DisabledSeriesIds,
			IncludeVariable:   body.Assumptions.IncludeVariable,
			IncludeUncertain:  body.Assumptions.IncludeUncertain,
		}
		req.Assumptions = &a
	}

	item, err := h.forecast.Create(r.Context(), req)
	if err != nil {
		writeForecastCreateError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPIForecast(item))
}

func (h *Handler) GetLatestForecast(w http.ResponseWriter, r *http.Request) {
	if h.forecast == nil {
		writeInternalError(w, "forecast service is not configured")
		return
	}
	item, err := h.forecast.LatestForCurrentBaseline(r.Context())
	if err != nil {
		if errors.Is(err, service.ErrForecastNotFound) || errors.Is(err, service.ErrBaselineNone) || errors.Is(err, service.ErrBaselineRequired) {
			writeNotFoundError(w, "no forecast available")
			return
		}
		writeInternalError(w, "failed to load forecast")
		return
	}
	writeJSON(w, http.StatusOK, toAPIForecast(item))
}

func (h *Handler) GetForecast(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	if h.forecast == nil {
		writeInternalError(w, "forecast service is not configured")
		return
	}
	item, err := h.forecast.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrForecastNotFound) {
			writeNotFoundError(w, err.Error())
			return
		}
		writeInternalError(w, "failed to load forecast")
		return
	}
	writeJSON(w, http.StatusOK, toAPIForecast(item))
}

func (h *Handler) GetForecastScenarios(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	if h.forecast == nil {
		writeInternalError(w, "forecast service is not configured")
		return
	}
	items, err := h.forecast.ListScenarios(r.Context(), id)
	if err != nil {
		writeInternalError(w, "failed to list scenarios")
		return
	}
	data := make([]gen.Scenario, len(items))
	for i, item := range items {
		data[i] = toAPIScenario(item)
	}
	writeJSON(w, http.StatusOK, gen.ScenarioListResponse{Data: data})
}

func (h *Handler) PostForecastScenario(w http.ResponseWriter, r *http.Request, id openapi_types.UUID) {
	if h.forecast == nil {
		writeInternalError(w, "forecast service is not configured")
		return
	}
	var body gen.ScenarioCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeValidationError(w, "invalid JSON body")
		return
	}
	params, err := parseScenarioParams(body.Params)
	if err != nil {
		writeValidationError(w, err.Error())
		return
	}
	item, err := h.forecast.RunScenario(r.Context(), service.RunScenarioRequest{
		ForecastID: id,
		Kind:       forecast.ScenarioKind(body.Kind),
		Params:     params,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForecastNotFound):
			writeNotFoundError(w, err.Error())
		case errors.Is(err, service.ErrInvalidScenario):
			writeValidationError(w, err.Error())
		default:
			writeInternalError(w, "failed to run scenario")
		}
		return
	}
	writeJSON(w, http.StatusOK, toAPIScenario(item))
}

func writeForecastCreateError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrBaselineRequired):
		writeValidationError(w, err.Error())
	case errors.Is(err, service.ErrBaselineNotFound), errors.Is(err, service.ErrBaselineNone):
		writeNotFoundError(w, err.Error())
	default:
		writeInternalError(w, "failed to create forecast")
	}
}

func toAPIForecast(f service.ForecastArtifact) gen.Forecast {
	points := make([]gen.ForecastPoint, len(f.Points))
	for i, p := range f.Points {
		points[i] = gen.ForecastPoint{
			Date:    openapi_types.Date{Time: dateOnly(p.Date)},
			Balance: p.Balance.StringFixed(2),
		}
	}
	options := make([]gen.ForecastSeriesOption, len(f.SeriesOptions))
	for i, o := range f.SeriesOptions {
		options[i] = gen.ForecastSeriesOption{
			Id:          o.ID,
			DisplayName: o.DisplayName,
			Kind:        o.Kind,
			Interval:    o.Interval,
			Amount:      o.Amount.StringFixed(2),
			Enabled:     o.Enabled,
		}
	}
	disabled := f.Assumptions.DisabledSeriesIDs
	if disabled == nil {
		disabled = []string{}
	}
	return gen.Forecast{
		Id:              f.ID,
		BaselineId:      f.BaselineID,
		HorizonDays:     f.HorizonDays,
		StartingBalance: f.StartingBalance.StringFixed(2),
		Assumptions: gen.ForecastAssumptions{
			DisabledSeriesIds: disabled,
			IncludeVariable:   f.Assumptions.IncludeVariable,
			IncludeUncertain:  f.Assumptions.IncludeUncertain,
		},
		Points:           points,
		MinBalance:       f.MinBalance.StringFixed(2),
		EndingBalance:    f.EndingBalance.StringFixed(2),
		AlgorithmVersion: f.AlgorithmVersion,
		SeriesOptions:    options,
		CreatedAt:        f.CreatedAt.UTC(),
	}
}

func toAPIScenario(s service.ScenarioArtifact) gen.Scenario {
	paramsMap := scenarioParamsToMap(s.Params)
	result := gen.ScenarioResult{
		Kind:               string(s.Result.Kind),
		MinBalance:         s.Result.MinBalance.StringFixed(2),
		EndingBalance:      s.Result.EndingBalance.StringFixed(2),
		FreeCashflowDelta:  s.Result.FreeCashflowDelta.StringFixed(2),
		BaselineMinBalance: s.Result.BaselineMinBalance.StringFixed(2),
		Notes:              s.Result.Notes,
	}
	if s.Result.GoalFeasible != nil {
		result.GoalFeasible = s.Result.GoalFeasible
	}
	if s.Result.ProjectedAtByDate != nil {
		v := s.Result.ProjectedAtByDate.StringFixed(2)
		result.ProjectedAtByDate = &v
	}
	return gen.Scenario{
		Id:         s.ID,
		ForecastId: s.ForecastID,
		Kind:       gen.ScenarioKind(s.Kind),
		Params:     paramsMap,
		Result:     result,
		Status:     gen.ScenarioStatus(s.Status),
		CreatedAt:  s.CreatedAt.UTC(),
	}
}

func parseScenarioParams(raw map[string]interface{}) (forecast.ScenarioParams, error) {
	var out forecast.ScenarioParams
	if raw == nil {
		return out, nil
	}
	if v, ok := raw["monthly_amount"].(string); ok && v != "" {
		d, err := decimal.NewFromString(v)
		if err != nil {
			return out, errors.New("invalid monthly_amount")
		}
		out.MonthlyAmount = &d
	}
	if v, ok := raw["monthly_income_delta"].(string); ok && v != "" {
		d, err := decimal.NewFromString(v)
		if err != nil {
			return out, errors.New("invalid monthly_income_delta")
		}
		out.MonthlyIncomeDelta = &d
	}
	if v, ok := raw["one_off_amount"].(string); ok && v != "" {
		d, err := decimal.NewFromString(v)
		if err != nil {
			return out, errors.New("invalid one_off_amount")
		}
		out.OneOffAmount = &d
	}
	if v, ok := raw["goal_amount"].(string); ok && v != "" {
		d, err := decimal.NewFromString(v)
		if err != nil {
			return out, errors.New("invalid goal_amount")
		}
		out.GoalAmount = &d
	}
	if v, ok := raw["months"].(float64); ok {
		m := int(v)
		out.Months = &m
	}
	if v, ok := raw["start_date"].(string); ok && v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return out, errors.New("invalid start_date")
		}
		out.StartDate = &t
	}
	if v, ok := raw["by_date"].(string); ok && v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return out, errors.New("invalid by_date")
		}
		out.ByDate = &t
	}
	return out, nil
}

func scenarioParamsToMap(p forecast.ScenarioParams) map[string]interface{} {
	out := map[string]interface{}{}
	if p.MonthlyAmount != nil {
		out["monthly_amount"] = p.MonthlyAmount.StringFixed(2)
	}
	if p.MonthlyIncomeDelta != nil {
		out["monthly_income_delta"] = p.MonthlyIncomeDelta.StringFixed(2)
	}
	if p.OneOffAmount != nil {
		out["one_off_amount"] = p.OneOffAmount.StringFixed(2)
	}
	if p.GoalAmount != nil {
		out["goal_amount"] = p.GoalAmount.StringFixed(2)
	}
	if p.Months != nil {
		out["months"] = *p.Months
	}
	if p.StartDate != nil {
		out["start_date"] = p.StartDate.Format("2006-01-02")
	}
	if p.ByDate != nil {
		out["by_date"] = p.ByDate.Format("2006-01-02")
	}
	return out
}
