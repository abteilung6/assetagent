package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/abteilung6/assetagent/internal/llm"
	"github.com/abteilung6/assetagent/internal/service"
)

const (
	baselineToolName    = "get_baseline"
	moneyReviewToolName = "get_money_review"
	forecastToolName    = "get_forecast"
)

type BaselineSource interface {
	Current(ctx context.Context) (service.ComputedBaseline, error)
}

type MoneyReviewSource interface {
	List(ctx context.Context, limit int) ([]service.MoneyReview, error)
}

type ForecastSource interface {
	LatestForCurrentBaseline(ctx context.Context) (service.ForecastArtifact, error)
}

type baselineToolResult struct {
	OK                      bool                   `json:"ok"`
	Available               bool                   `json:"available"`
	ID                      string                 `json:"id,omitempty"`
	Status                  string                 `json:"status,omitempty"`
	Period                  period                 `json:"period,omitempty"`
	RegularMonthlyIncome    string                 `json:"regular_monthly_income,omitempty"`
	MonthlyFixedCosts       string                 `json:"monthly_fixed_costs,omitempty"`
	MonthlyIrregularCosts   string                 `json:"monthly_irregular_costs,omitempty"`
	AvgVariableSpend        string                 `json:"avg_variable_spend,omitempty"`
	SustainableFreeCashflow string                 `json:"sustainable_free_cashflow,omitempty"`
	Currency                string                 `json:"currency,omitempty"`
	Confidence              string                 `json:"confidence,omitempty"`
	Assumptions             []string               `json:"assumptions,omitempty"`
	Metrics                 []baselineMetricResult `json:"metrics,omitempty"`
	Calculation             string                 `json:"calculation,omitempty"`
	EvidenceIDs             []string               `json:"evidence_ids,omitempty"`
	Message                 string                 `json:"message,omitempty"`
}

type baselineMetricResult struct {
	Key         string   `json:"key"`
	Value       string   `json:"value"`
	Calculation string   `json:"calculation"`
	Confidence  string   `json:"confidence"`
	EvidenceIDs []string `json:"evidence_ids"`
}

type moneyReviewToolResult struct {
	OK            bool                       `json:"ok"`
	Available     bool                       `json:"available"`
	ID            string                     `json:"id,omitempty"`
	BaselineID    string                     `json:"baseline_id,omitempty"`
	Status        string                     `json:"status,omitempty"`
	Period        period                     `json:"period,omitempty"`
	Summary       string                     `json:"summary,omitempty"`
	Findings      []moneyReviewFindingResult `json:"findings,omitempty"`
	DataFreshness string                     `json:"data_freshness,omitempty"`
	Calculation   string                     `json:"calculation,omitempty"`
	EvidenceIDs   []string                   `json:"evidence_ids,omitempty"`
	Message       string                     `json:"message,omitempty"`
}

type moneyReviewFindingResult struct {
	Type               string   `json:"type"`
	Title              string   `json:"title"`
	Amount             *string  `json:"amount,omitempty"`
	Confidence         string   `json:"confidence"`
	EvidenceIDs        []string `json:"evidence_ids"`
	SuggestedActionKey string   `json:"suggested_action_key,omitempty"`
}

type forecastToolResult struct {
	OK               bool     `json:"ok"`
	Available        bool     `json:"available"`
	ID               string   `json:"id,omitempty"`
	BaselineID       string   `json:"baseline_id,omitempty"`
	HorizonDays      int      `json:"horizon_days,omitempty"`
	StartingBalance  string   `json:"starting_balance,omitempty"`
	MinBalance       string   `json:"min_balance,omitempty"`
	EndingBalance    string   `json:"ending_balance,omitempty"`
	Currency         string   `json:"currency,omitempty"`
	AlgorithmVersion string   `json:"algorithm_version,omitempty"`
	Assumptions      []string `json:"assumptions,omitempty"`
	Calculation      string   `json:"calculation,omitempty"`
	Confidence       string   `json:"confidence,omitempty"`
	EvidenceIDs      []string `json:"evidence_ids,omitempty"`
	Message          string   `json:"message,omitempty"`
}

func baselineTool(source BaselineSource) toolEntry {
	return toolEntry{
		definition: llm.Tool{
			Name:        baselineToolName,
			Description: "Get the current FinancialBaseline (income, fixed/irregular/variable costs, sustainable free cashflow). Prefer this for plan / monthly budget questions. Read-only — do not invent numbers.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
		handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			return runGetBaseline(ctx, source)
		},
	}
}

func moneyReviewTool(source MoneyReviewSource) toolEntry {
	return toolEntry{
		definition: llm.Tool{
			Name:        moneyReviewToolName,
			Description: "Get the latest Money Review summary and findings pinned to a baseline. Prefer this when explaining the monthly review. Read-only.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
		handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			return runGetMoneyReview(ctx, source)
		},
	}
}

func forecastTool(source ForecastSource) toolEntry {
	return toolEntry{
		definition: llm.Tool{
			Name:        forecastToolName,
			Description: "Get the latest 90-day liquidity forecast summary (starting/min/ending balance) for the current baseline. Prefer this for runway / cash projection questions. Read-only — scenarios are run in the Plan UI.",
			Parameters: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
		handler: func(ctx context.Context, raw json.RawMessage) (any, error) {
			return runGetForecast(ctx, source)
		},
	}
}

func runGetBaseline(ctx context.Context, source BaselineSource) (baselineToolResult, error) {
	if source == nil {
		return baselineToolResult{}, errors.New("baseline service is not configured")
	}

	baseline, err := source.Current(ctx)
	if err != nil {
		if errors.Is(err, service.ErrBaselineNone) {
			return baselineToolResult{
				OK:        true,
				Available: false,
				Message:   "No baseline available yet. Compute and confirm one on the Baseline page.",
			}, nil
		}
		return baselineToolResult{}, err
	}

	metrics := make([]baselineMetricResult, 0, len(baseline.Metrics))
	evidenceIDs := make([]string, 0)
	for _, m := range baseline.Metrics {
		ids := m.EvidenceIDs
		if ids == nil {
			ids = []string{}
		}
		metrics = append(metrics, baselineMetricResult{
			Key:         m.Key,
			Value:       decimalString(m.Value),
			Calculation: m.Calculation,
			Confidence:  m.Confidence,
			EvidenceIDs: ids,
		})
		evidenceIDs = append(evidenceIDs, ids...)
	}
	assumptions := baseline.Assumptions
	if assumptions == nil {
		assumptions = []string{}
	}
	if evidenceIDs == nil {
		evidenceIDs = []string{}
	}

	return baselineToolResult{
		OK:                      true,
		Available:               true,
		ID:                      baseline.ID.String(),
		Status:                  baseline.Status,
		Period:                  periodFromTimes(baseline.PeriodFrom, baseline.PeriodTo),
		RegularMonthlyIncome:    decimalString(baseline.RegularMonthlyIncome),
		MonthlyFixedCosts:       decimalString(baseline.MonthlyFixedCosts),
		MonthlyIrregularCosts:   decimalString(baseline.MonthlyIrregularCosts),
		AvgVariableSpend:        decimalString(baseline.AvgVariableSpend),
		SustainableFreeCashflow: decimalString(baseline.SustainableFreeCashflow),
		Currency:                "EUR",
		Confidence:              baseline.Confidence,
		Assumptions:             assumptions,
		Metrics:                 metrics,
		Calculation:             "FinancialBaseline from recurring series and transfer-aware residual spend",
		EvidenceIDs:             evidenceIDs,
	}, nil
}

func runGetMoneyReview(ctx context.Context, source MoneyReviewSource) (moneyReviewToolResult, error) {
	if source == nil {
		return moneyReviewToolResult{}, errors.New("money review service is not configured")
	}

	reviews, err := source.List(ctx, 1)
	if err != nil {
		return moneyReviewToolResult{}, err
	}
	if len(reviews) == 0 {
		return moneyReviewToolResult{
			OK:        true,
			Available: false,
			Message:   "No money review yet. Create one on the Reviews page after confirming a baseline.",
			Findings:  []moneyReviewFindingResult{},
		}, nil
	}

	review := reviews[0]
	findings := make([]moneyReviewFindingResult, 0, len(review.Findings))
	evidenceIDs := make([]string, 0)
	for _, f := range review.Findings {
		ids := f.EvidenceIDs
		if ids == nil {
			ids = []string{}
		}
		var amount *string
		if f.Amount != nil {
			s := decimalString(*f.Amount)
			amount = &s
		}
		findings = append(findings, moneyReviewFindingResult{
			Type:               f.Type,
			Title:              f.Title,
			Amount:             amount,
			Confidence:         f.Confidence,
			EvidenceIDs:        ids,
			SuggestedActionKey: f.SuggestedActionKey,
		})
		evidenceIDs = append(evidenceIDs, ids...)
	}
	if evidenceIDs == nil {
		evidenceIDs = []string{}
	}

	return moneyReviewToolResult{
		OK:            true,
		Available:     true,
		ID:            review.ID.String(),
		BaselineID:    review.BaselineID.String(),
		Status:        review.Status,
		Period:        periodFromTimes(review.PeriodFrom, review.PeriodTo),
		Summary:       review.Summary,
		Findings:      findings,
		DataFreshness: review.DataFreshness,
		Calculation:   "Money Review findings generated from baseline and trusted money facts",
		EvidenceIDs:   evidenceIDs,
	}, nil
}

func runGetForecast(ctx context.Context, source ForecastSource) (forecastToolResult, error) {
	if source == nil {
		return forecastToolResult{}, errors.New("forecast service is not configured")
	}

	artifact, err := source.LatestForCurrentBaseline(ctx)
	if err != nil {
		if errors.Is(err, service.ErrForecastNotFound) || errors.Is(err, service.ErrBaselineNone) {
			return forecastToolResult{
				OK:        true,
				Available: false,
				Message:   "No forecast available yet. Run one from the Plan page after confirming a baseline.",
			}, nil
		}
		return forecastToolResult{}, err
	}

	assumptions := []string{
		fmt.Sprintf("horizon_days=%d", artifact.HorizonDays),
		fmt.Sprintf("include_variable=%v", artifact.Assumptions.IncludeVariable),
		fmt.Sprintf("include_uncertain=%v", artifact.Assumptions.IncludeUncertain),
	}
	if len(artifact.Assumptions.DisabledSeriesIDs) > 0 {
		assumptions = append(assumptions, fmt.Sprintf("disabled_series=%d", len(artifact.Assumptions.DisabledSeriesIDs)))
	}

	return forecastToolResult{
		OK:               true,
		Available:        true,
		ID:               artifact.ID.String(),
		BaselineID:       artifact.BaselineID.String(),
		HorizonDays:      artifact.HorizonDays,
		StartingBalance:  decimalString(artifact.StartingBalance),
		MinBalance:       decimalString(artifact.MinBalance),
		EndingBalance:    decimalString(artifact.EndingBalance),
		Currency:         "EUR",
		AlgorithmVersion: artifact.AlgorithmVersion,
		Assumptions:      assumptions,
		Calculation:      "Deterministic 90-day liquidity projection from baseline and recurring series",
		Confidence:       "medium",
		EvidenceIDs:      []string{"forecast:" + artifact.ID.String(), "baseline:" + artifact.BaselineID.String()},
	}, nil
}

func periodFromTimes(from, to time.Time) period {
	return period{
		From: from.UTC().Format("2006-01-02"),
		To:   to.UTC().Format("2006-01-02"),
	}
}
