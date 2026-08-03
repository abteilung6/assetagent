package service

import (
	"testing"

	"github.com/abteilung6/assetagent/internal/finance"
	"github.com/shopspring/decimal"
)

func TestApplyMetricAdjustment_recalculatesFreeCashflow(t *testing.T) {
	b := ComputedBaseline{
		RegularMonthlyIncome:    decimal.RequireFromString("3500.00"),
		MonthlyFixedCosts:       decimal.RequireFromString("1200.00"),
		MonthlyIrregularCosts:   decimal.RequireFromString("50.00"),
		AvgVariableSpend:        decimal.RequireFromString("450.00"),
		SustainableFreeCashflow: decimal.RequireFromString("1800.00"),
		Metrics: []finance.MetricEvidence{
			{Key: finance.MetricMonthlyFixedCosts, Value: decimal.RequireFromString("1200.00")},
			{Key: finance.MetricSustainableFreeCash, Value: decimal.RequireFromString("1800.00")},
		},
	}

	got, err := applyMetricAdjustment(b, finance.MetricMonthlyFixedCosts, decimal.RequireFromString("1100.00"))
	if err != nil {
		t.Fatal(err)
	}
	if !got.MonthlyFixedCosts.Equal(decimal.RequireFromString("1100.00")) {
		t.Fatalf("fixed = %s", got.MonthlyFixedCosts)
	}
	// 3500 - 1100 - 50 - 450 = 1900
	if !got.SustainableFreeCashflow.Equal(decimal.RequireFromString("1900.00")) {
		t.Fatalf("free = %s", got.SustainableFreeCashflow)
	}
}

func TestApplyMetricAdjustment_rejectsUnknownKey(t *testing.T) {
	_, err := applyMetricAdjustment(ComputedBaseline{}, "not_a_metric", decimal.Zero)
	if err != ErrInvalidBaselineMetric {
		t.Fatalf("err = %v", err)
	}
}
