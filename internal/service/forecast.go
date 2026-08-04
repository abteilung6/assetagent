package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sqldb "github.com/abteilung6/assetagent/internal/db/sqlc"
	"github.com/abteilung6/assetagent/internal/forecast"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

var (
	ErrForecastNotFound = errors.New("forecast not found")
	ErrInvalidScenario  = errors.New("invalid scenario")
)

// ForecastArtifact is a persisted projection.
type ForecastArtifact struct {
	ID               uuid.UUID
	BaselineID       uuid.UUID
	HorizonDays      int
	StartingBalance  decimal.Decimal
	Assumptions      forecast.Assumptions
	Points           []forecast.Point
	MinBalance       decimal.Decimal
	EndingBalance    decimal.Decimal
	AlgorithmVersion string
	CreatedAt        time.Time
	SeriesOptions    []ForecastSeriesOption
}

// ForecastSeriesOption is a toggleable recurring series for the UI.
type ForecastSeriesOption struct {
	ID          string
	DisplayName string
	Kind        string
	Interval    string
	Amount      decimal.Decimal
	Enabled     bool
}

// ScenarioArtifact is a persisted typed scenario run.
type ScenarioArtifact struct {
	ID         uuid.UUID
	ForecastID uuid.UUID
	Kind       forecast.ScenarioKind
	Params     forecast.ScenarioParams
	Result     forecast.ScenarioResult
	Status     string
	CreatedAt  time.Time
}

// ForecastService creates forecasts and scenarios.
type ForecastService struct {
	pool     *pgxpool.Pool
	baseline *BaselineService
	now      func() time.Time
}

func NewForecast(pool *pgxpool.Pool) *ForecastService {
	return &ForecastService{
		pool:     pool,
		baseline: NewBaseline(pool),
		now:      func() time.Time { return time.Now().UTC() },
	}
}

// NewForecastAt is like NewForecast but freezes "now" for deterministic projections (golden/evals).
func NewForecastAt(pool *pgxpool.Pool, at time.Time) *ForecastService {
	at = time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC)
	return &ForecastService{
		pool:     pool,
		baseline: NewBaseline(pool),
		now:      func() time.Time { return at },
	}
}

type CreateForecastRequest struct {
	BaselineID      *uuid.UUID
	StartingBalance decimal.Decimal
	HorizonDays     int
	Assumptions     *forecast.Assumptions
}

func (s *ForecastService) Create(ctx context.Context, req CreateForecastRequest) (ForecastArtifact, error) {
	baseline, err := s.resolveBaseline(ctx, req.BaselineID)
	if err != nil {
		return ForecastArtifact{}, err
	}

	seriesRows, err := NewRecurring(s.pool).List(ctx)
	if err != nil {
		return ForecastArtifact{}, fmt.Errorf("list recurring: %w", err)
	}
	series := forecast.SeriesFromDomain(seriesRows)

	assumptions := forecast.DefaultAssumptions()
	if req.Assumptions != nil {
		assumptions = *req.Assumptions
		if assumptions.DisabledSeriesIDs == nil {
			assumptions.DisabledSeriesIDs = []string{}
		}
	}

	horizon := req.HorizonDays
	if horizon <= 0 {
		horizon = forecast.DefaultHorizonDays
	}

	result := forecast.Project(forecast.Input{
		StartingBalance: req.StartingBalance,
		StartDate:       s.now(),
		HorizonDays:     horizon,
		Series:          series,
		VariableMonthly: baseline.AvgVariableSpend,
		Assumptions:     assumptions,
	})

	disabled := make(map[string]struct{}, len(assumptions.DisabledSeriesIDs))
	for _, id := range assumptions.DisabledSeriesIDs {
		disabled[id] = struct{}{}
	}
	options := make([]ForecastSeriesOption, 0, len(series))
	for _, item := range series {
		if item.Status == "ended" {
			continue
		}
		_, isDisabled := disabled[item.ID]
		options = append(options, ForecastSeriesOption{
			ID:          item.ID,
			DisplayName: item.DisplayName,
			Kind:        item.Kind,
			Interval:    item.Interval,
			Amount:      item.AmountTypical,
			Enabled:     !isDisabled,
		})
	}

	artifact := ForecastArtifact{
		BaselineID:       baseline.ID,
		HorizonDays:      result.HorizonDays,
		StartingBalance:  result.StartingBalance,
		Assumptions:      result.Assumptions,
		Points:           result.Points,
		MinBalance:       result.MinBalance,
		EndingBalance:    result.EndingBalance,
		AlgorithmVersion: result.AlgorithmVersion,
		SeriesOptions:    options,
	}
	return s.insertForecast(ctx, artifact)
}

func (s *ForecastService) Get(ctx context.Context, id uuid.UUID) (ForecastArtifact, error) {
	householdID, err := repository.ResolveHouseholdID(ctx, s.pool)
	if err != nil {
		return ForecastArtifact{}, err
	}
	row, err := sqldb.New(s.pool).GetForecast(ctx, sqldb.GetForecastParams{
		ID:          id,
		HouseholdID: householdID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ForecastArtifact{}, ErrForecastNotFound
		}
		return ForecastArtifact{}, err
	}
	return mapForecastRow(row)
}

func (s *ForecastService) LatestForCurrentBaseline(ctx context.Context) (ForecastArtifact, error) {
	baseline, err := s.baseline.Current(ctx)
	if err != nil {
		return ForecastArtifact{}, err
	}
	householdID, err := repository.ResolveHouseholdID(ctx, s.pool)
	if err != nil {
		return ForecastArtifact{}, err
	}
	row, err := sqldb.New(s.pool).GetLatestForecastForBaseline(ctx, sqldb.GetLatestForecastForBaselineParams{
		BaselineID:  baseline.ID,
		HouseholdID: householdID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ForecastArtifact{}, ErrForecastNotFound
		}
		return ForecastArtifact{}, err
	}
	return mapForecastRow(row)
}

type RunScenarioRequest struct {
	ForecastID uuid.UUID
	Kind       forecast.ScenarioKind
	Params     forecast.ScenarioParams
}

func (s *ForecastService) RunScenario(ctx context.Context, req RunScenarioRequest) (ScenarioArtifact, error) {
	artifact, err := s.Get(ctx, req.ForecastID)
	if err != nil {
		return ScenarioArtifact{}, err
	}
	baseline, err := s.resolveBaseline(ctx, &artifact.BaselineID)
	if err != nil {
		return ScenarioArtifact{}, err
	}
	seriesRows, err := NewRecurring(s.pool).List(ctx)
	if err != nil {
		return ScenarioArtifact{}, err
	}

	baseInput := forecast.Input{
		StartingBalance: artifact.StartingBalance,
		StartDate:       artifact.CreatedAt,
		HorizonDays:     artifact.HorizonDays,
		Series:          forecast.SeriesFromDomain(seriesRows),
		VariableMonthly: baseline.AvgVariableSpend,
		Assumptions:     artifact.Assumptions,
	}

	result, err := forecast.ApplyScenario(baseInput, req.Kind, req.Params)
	if err != nil {
		return ScenarioArtifact{}, fmt.Errorf("%w: %v", ErrInvalidScenario, err)
	}

	paramsJSON, err := json.Marshal(req.Params)
	if err != nil {
		return ScenarioArtifact{}, err
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return ScenarioArtifact{}, err
	}

	householdID, err := repository.ResolveHouseholdID(ctx, s.pool)
	if err != nil {
		return ScenarioArtifact{}, err
	}
	row, err := sqldb.New(s.pool).InsertScenario(ctx, sqldb.InsertScenarioParams{
		HouseholdID: householdID,
		ForecastID:  req.ForecastID,
		Kind:        string(req.Kind),
		Params:      paramsJSON,
		Result:      resultJSON,
		Status:      "confirmed",
	})
	if err != nil {
		return ScenarioArtifact{}, fmt.Errorf("insert scenario: %w", err)
	}
	return ScenarioArtifact{
		ID:         row.ID,
		ForecastID: row.ForecastID,
		Kind:       req.Kind,
		Params:     req.Params,
		Result:     result,
		Status:     row.Status,
		CreatedAt:  row.CreatedAt.Time,
	}, nil
}

func (s *ForecastService) ListScenarios(ctx context.Context, forecastID uuid.UUID) ([]ScenarioArtifact, error) {
	householdID, err := repository.ResolveHouseholdID(ctx, s.pool)
	if err != nil {
		return nil, err
	}
	rows, err := sqldb.New(s.pool).ListScenariosForForecast(ctx, sqldb.ListScenariosForForecastParams{
		ForecastID:  forecastID,
		HouseholdID: householdID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]ScenarioArtifact, 0, len(rows))
	for _, row := range rows {
		item, err := mapScenarioRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *ForecastService) resolveBaseline(ctx context.Context, id *uuid.UUID) (ComputedBaseline, error) {
	if id == nil {
		b, err := s.baseline.Current(ctx)
		if err != nil {
			if errors.Is(err, ErrBaselineNone) {
				return ComputedBaseline{}, ErrBaselineRequired
			}
			return ComputedBaseline{}, err
		}
		return b, nil
	}
	householdID, err := repository.ResolveHouseholdID(ctx, s.pool)
	if err != nil {
		return ComputedBaseline{}, err
	}
	row, err := sqldb.New(s.pool).GetFinancialBaseline(ctx, sqldb.GetFinancialBaselineParams{
		ID:          *id,
		HouseholdID: householdID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ComputedBaseline{}, ErrBaselineNotFound
		}
		return ComputedBaseline{}, err
	}
	return mapBaselineRow(row)
}

func (s *ForecastService) insertForecast(ctx context.Context, artifact ForecastArtifact) (ForecastArtifact, error) {
	assumptionsJSON, err := json.Marshal(artifact.Assumptions)
	if err != nil {
		return ForecastArtifact{}, err
	}
	seriesJSON, err := pointsToJSON(artifact.Points)
	if err != nil {
		return ForecastArtifact{}, err
	}
	// Also embed series options in assumptions payload for UI reload.
	type storedAssumptions struct {
		forecast.Assumptions
		SeriesOptions []ForecastSeriesOption `json:"series_options"`
	}
	stored := storedAssumptions{
		Assumptions:   artifact.Assumptions,
		SeriesOptions: artifact.SeriesOptions,
	}
	assumptionsJSON, err = json.Marshal(stored)
	if err != nil {
		return ForecastArtifact{}, err
	}

	householdID, err := repository.ResolveHouseholdID(ctx, s.pool)
	if err != nil {
		return ForecastArtifact{}, err
	}
	row, err := sqldb.New(s.pool).InsertForecast(ctx, sqldb.InsertForecastParams{
		HouseholdID:      householdID,
		BaselineID:       artifact.BaselineID,
		HorizonDays:      int32(artifact.HorizonDays),
		StartingBalance:  artifact.StartingBalance,
		Assumptions:      assumptionsJSON,
		Series:           seriesJSON,
		MinBalance:       artifact.MinBalance,
		EndingBalance:    artifact.EndingBalance,
		AlgorithmVersion: artifact.AlgorithmVersion,
	})
	if err != nil {
		return ForecastArtifact{}, fmt.Errorf("insert forecast: %w", err)
	}
	return mapForecastRow(row)
}

func pointsToJSON(points []forecast.Point) ([]byte, error) {
	type wire struct {
		Date    string `json:"date"`
		Balance string `json:"balance"`
	}
	out := make([]wire, len(points))
	for i, p := range points {
		out[i] = wire{Date: p.Date.Format("2006-01-02"), Balance: p.Balance.StringFixed(2)}
	}
	return json.Marshal(out)
}

func mapForecastRow(row sqldb.Forecast) (ForecastArtifact, error) {
	points, err := pointsFromJSON(row.Series)
	if err != nil {
		return ForecastArtifact{}, err
	}
	assumptions, options, err := assumptionsFromJSON(row.Assumptions)
	if err != nil {
		return ForecastArtifact{}, err
	}
	return ForecastArtifact{
		ID:               row.ID,
		BaselineID:       row.BaselineID,
		HorizonDays:      int(row.HorizonDays),
		StartingBalance:  row.StartingBalance,
		Assumptions:      assumptions,
		Points:           points,
		MinBalance:       row.MinBalance,
		EndingBalance:    row.EndingBalance,
		AlgorithmVersion: row.AlgorithmVersion,
		CreatedAt:        row.CreatedAt.Time,
		SeriesOptions:    options,
	}, nil
}

func assumptionsFromJSON(raw []byte) (forecast.Assumptions, []ForecastSeriesOption, error) {
	type stored struct {
		DisabledSeriesIDs []string               `json:"disabled_series_ids"`
		IncludeVariable   bool                   `json:"include_variable"`
		IncludeUncertain  bool                   `json:"include_uncertain"`
		SeriesOptions     []ForecastSeriesOption `json:"series_options"`
	}
	var s stored
	if err := json.Unmarshal(raw, &s); err != nil {
		return forecast.Assumptions{}, nil, err
	}
	a := forecast.Assumptions{
		DisabledSeriesIDs: s.DisabledSeriesIDs,
		IncludeVariable:   s.IncludeVariable,
		IncludeUncertain:  s.IncludeUncertain,
	}
	if a.DisabledSeriesIDs == nil {
		a.DisabledSeriesIDs = []string{}
	}
	return a, s.SeriesOptions, nil
}

func pointsFromJSON(raw []byte) ([]forecast.Point, error) {
	type wire struct {
		Date    string `json:"date"`
		Balance string `json:"balance"`
	}
	var items []wire
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	out := make([]forecast.Point, len(items))
	for i, item := range items {
		d, err := time.Parse("2006-01-02", item.Date)
		if err != nil {
			return nil, err
		}
		bal, err := decimal.NewFromString(item.Balance)
		if err != nil {
			return nil, err
		}
		out[i] = forecast.Point{Date: d, Balance: bal}
	}
	return out, nil
}

func mapScenarioRow(row sqldb.Scenario) (ScenarioArtifact, error) {
	var params forecast.ScenarioParams
	if err := json.Unmarshal(row.Params, &params); err != nil {
		return ScenarioArtifact{}, err
	}
	var result forecast.ScenarioResult
	if err := json.Unmarshal(row.Result, &result); err != nil {
		return ScenarioArtifact{}, err
	}
	return ScenarioArtifact{
		ID:         row.ID,
		ForecastID: row.ForecastID,
		Kind:       forecast.ScenarioKind(row.Kind),
		Params:     params,
		Result:     result,
		Status:     row.Status,
		CreatedAt:  row.CreatedAt.Time,
	}, nil
}
