package tools_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/abteilung6/assetagent/internal/chat/tools"
	"github.com/abteilung6/assetagent/internal/finance"
	"github.com/abteilung6/assetagent/internal/forecast"
	"github.com/abteilung6/assetagent/internal/review"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type fakeBaseline struct {
	baseline service.ComputedBaseline
	err      error
}

func (f *fakeBaseline) Current(ctx context.Context) (service.ComputedBaseline, error) {
	return f.baseline, f.err
}

type fakeMoneyReview struct {
	reviews []service.MoneyReview
	err     error
}

func (f *fakeMoneyReview) List(ctx context.Context, limit int) ([]service.MoneyReview, error) {
	if f.err != nil {
		return nil, f.err
	}
	if limit > 0 && limit < len(f.reviews) {
		return f.reviews[:limit], nil
	}
	return f.reviews, nil
}

type fakeForecast struct {
	artifact service.ForecastArtifact
	err      error
}

func (f *fakeForecast) LatestForCurrentBaseline(ctx context.Context) (service.ForecastArtifact, error) {
	return f.artifact, f.err
}

func TestRegistry_executePlanArtifacts(t *testing.T) {
	baselineID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	reviewID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	forecastID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	amount := decimal.RequireFromString("-120.00")

	registry := tools.NewRegistry(tools.Dependencies{
		Reports: &fakeReports{},
		Lister:  &fakeLister{},
		Baseline: &fakeBaseline{
			baseline: service.ComputedBaseline{
				ID:                      baselineID,
				PeriodFrom:              time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
				PeriodTo:                time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
				Status:                  service.BaselineStatusConfirmed,
				RegularMonthlyIncome:    decimal.RequireFromString("3500.00"),
				MonthlyFixedCosts:       decimal.RequireFromString("1200.00"),
				MonthlyIrregularCosts:   decimal.RequireFromString("50.00"),
				AvgVariableSpend:        decimal.RequireFromString("450.00"),
				SustainableFreeCashflow: decimal.RequireFromString("1800.00"),
				Confidence:              finance.ConfidenceHigh,
				Assumptions:             []string{"period=last_complete_calendar_month"},
				Metrics: []finance.MetricEvidence{{
					Key:         finance.MetricSustainableFreeCash,
					Value:       decimal.RequireFromString("1800.00"),
					Calculation: "income − costs",
					Confidence:  finance.ConfidenceHigh,
					EvidenceIDs: []string{"baseline_free_cashflow"},
				}},
			},
		},
		MoneyReview: &fakeMoneyReview{
			reviews: []service.MoneyReview{{
				ID:         reviewID,
				BaselineID: baselineID,
				PeriodFrom: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
				PeriodTo:   time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
				Status:     service.MoneyReviewStatusConfirmed,
				Summary:    "Free cashflow looks healthy.",
				Findings: []review.Finding{{
					Type:               review.FindingLargeExpense,
					Title:              "Large furniture purchase",
					Amount:             &amount,
					Confidence:         review.ConfidenceHigh,
					EvidenceIDs:        []string{"tx:1"},
					SuggestedActionKey: "one_off_ok",
				}},
				DataFreshness: "2026-04-01",
			}},
		},
		Forecast: &fakeForecast{
			artifact: service.ForecastArtifact{
				ID:               forecastID,
				BaselineID:       baselineID,
				HorizonDays:      90,
				StartingBalance:  decimal.RequireFromString("5000.00"),
				MinBalance:       decimal.RequireFromString("3200.00"),
				EndingBalance:    decimal.RequireFromString("4100.00"),
				AlgorithmVersion: "forecast.v1",
				Assumptions:      forecast.DefaultAssumptions(),
			},
		},
	})

	raw, err := registry.Execute(context.Background(), "get_baseline", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("get_baseline: %v", err)
	}
	var baseline map[string]any
	if err := json.Unmarshal(raw, &baseline); err != nil {
		t.Fatalf("decode baseline: %v", err)
	}
	if baseline["ok"] != true || baseline["available"] != true {
		t.Fatalf("baseline = %#v", baseline)
	}
	if baseline["sustainable_free_cashflow"] != "1800" && baseline["sustainable_free_cashflow"] != "1800.00" {
		// decimal.String() may omit trailing zeros
		if s, _ := baseline["sustainable_free_cashflow"].(string); s != "1800" && s != "1800.00" {
			t.Fatalf("free cashflow = %v", baseline["sustainable_free_cashflow"])
		}
	}

	raw, err = registry.Execute(context.Background(), "get_money_review", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("get_money_review: %v", err)
	}
	var moneyReview map[string]any
	if err := json.Unmarshal(raw, &moneyReview); err != nil {
		t.Fatalf("decode money review: %v", err)
	}
	if moneyReview["ok"] != true || moneyReview["available"] != true {
		t.Fatalf("money review = %#v", moneyReview)
	}
	if moneyReview["summary"] != "Free cashflow looks healthy." {
		t.Fatalf("summary = %v", moneyReview["summary"])
	}

	raw, err = registry.Execute(context.Background(), "get_forecast", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("get_forecast: %v", err)
	}
	var forecastResult map[string]any
	if err := json.Unmarshal(raw, &forecastResult); err != nil {
		t.Fatalf("decode forecast: %v", err)
	}
	if forecastResult["ok"] != true || forecastResult["available"] != true {
		t.Fatalf("forecast = %#v", forecastResult)
	}
	if forecastResult["min_balance"] != "3200" && forecastResult["min_balance"] != "3200.00" {
		if s, _ := forecastResult["min_balance"].(string); s != "3200" && s != "3200.00" {
			t.Fatalf("min_balance = %v", forecastResult["min_balance"])
		}
	}
}

func TestRegistry_planArtifactsUnavailable(t *testing.T) {
	registry := tools.NewRegistry(tools.Dependencies{
		Reports:     &fakeReports{},
		Lister:      &fakeLister{},
		Baseline:    &fakeBaseline{err: service.ErrBaselineNone},
		MoneyReview: &fakeMoneyReview{reviews: nil},
		Forecast:    &fakeForecast{err: service.ErrForecastNotFound},
	})

	for _, name := range []string{"get_baseline", "get_money_review", "get_forecast"} {
		raw, err := registry.Execute(context.Background(), name, json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var result map[string]any
		if err := json.Unmarshal(raw, &result); err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}
		if result["ok"] != true || result["available"] != false {
			t.Fatalf("%s = %#v", name, result)
		}
	}
}
