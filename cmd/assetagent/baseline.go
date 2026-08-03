package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/abteilung6/assetagent/internal/finance"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func newBaselineCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "baseline",
		Short: "Developer tools for FinancialBaseline (not an end-user path)",
	}
	cmd.AddCommand(newBaselineRecomputeCmd())
	cmd.AddCommand(newBaselineCurrentCmd())
	cmd.AddCommand(newBaselineConfirmCmd())
	return cmd
}

func newBaselineRecomputeCmd() *cobra.Command {
	var fromStr, toStr string
	var asJSON bool
	var save bool

	cmd := &cobra.Command{
		Use:   "recompute",
		Short: "Compute FinancialBaseline from trusted money facts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pool, cleanup, err := openImportDB()
			if err != nil {
				return err
			}
			defer cleanup()

			var from, to *time.Time
			if fromStr != "" || toStr != "" {
				if fromStr == "" || toStr == "" {
					return fmt.Errorf("--from and --to must both be set")
				}
				f, err := time.Parse("2006-01-02", fromStr)
				if err != nil {
					return fmt.Errorf("parse --from: %w", err)
				}
				t, err := time.Parse("2006-01-02", toStr)
				if err != nil {
					return fmt.Errorf("parse --to: %w", err)
				}
				from, to = &f, &t
			}

			svc := service.NewBaseline(pool)
			var baseline service.ComputedBaseline
			if save {
				baseline, err = svc.RecomputeAndSave(context.Background(), from, to)
			} else {
				baseline, err = svc.Recompute(context.Background(), from, to)
			}
			if err != nil {
				return err
			}

			if asJSON {
				return printBaselineJSON(baseline)
			}
			printBaselineTable(baseline)
			return nil
		},
	}

	cmd.Flags().StringVar(&fromStr, "from", "", "Period start YYYY-MM-DD (optional)")
	cmd.Flags().StringVar(&toStr, "to", "", "Period end YYYY-MM-DD (optional)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Print JSON instead of a table")
	cmd.Flags().BoolVar(&save, "save", false, "Persist as a new draft baseline")
	return cmd
}

func newBaselineCurrentCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "current",
		Short: "Show the current draft or confirmed baseline",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pool, cleanup, err := openImportDB()
			if err != nil {
				return err
			}
			defer cleanup()

			baseline, err := service.NewBaseline(pool).Current(context.Background())
			if err != nil {
				return err
			}
			if asJSON {
				return printBaselineJSON(baseline)
			}
			printBaselineTable(baseline)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Print JSON instead of a table")
	return cmd
}

func newBaselineConfirmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "confirm [id]",
		Short: "Confirm a draft baseline",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("parse id: %w", err)
			}
			pool, cleanup, err := openImportDB()
			if err != nil {
				return err
			}
			defer cleanup()

			baseline, err := service.NewBaseline(pool).Confirm(context.Background(), id)
			if err != nil {
				return err
			}
			printBaselineTable(baseline)
			return nil
		},
	}
}

func printBaselineTable(b service.ComputedBaseline) {
	fmt.Println("FinancialBaseline")
	if b.ID != uuid.Nil {
		fmt.Printf("  ID:         %s\n", b.ID)
	}
	fmt.Printf("  Period:     %s → %s\n", b.PeriodFrom.Format("2006-01-02"), b.PeriodTo.Format("2006-01-02"))
	fmt.Printf("  Algorithm:  %s\n", b.AlgorithmVersion)
	fmt.Printf("  Confidence: %s\n", b.Confidence)
	fmt.Printf("  Status:     %s\n", b.Status)
	fmt.Println()
	fmt.Printf("  %-28s  %10s\n", "Metric", "EUR/mo")
	fmt.Printf("  %-28s  %10s\n", "----------------------------", "----------")
	fmt.Printf("  %-28s  %10s\n", finance.MetricRegularMonthlyIncome, b.RegularMonthlyIncome.StringFixed(2))
	fmt.Printf("  %-28s  %10s\n", finance.MetricMonthlyFixedCosts, b.MonthlyFixedCosts.StringFixed(2))
	fmt.Printf("  %-28s  %10s\n", finance.MetricMonthlyIrregularCosts, b.MonthlyIrregularCosts.StringFixed(2))
	fmt.Printf("  %-28s  %10s\n", finance.MetricAvgVariableSpend, b.AvgVariableSpend.StringFixed(2))
	fmt.Printf("  %-28s  %10s\n", finance.MetricSustainableFreeCash, b.SustainableFreeCashflow.StringFixed(2))
	if len(b.Assumptions) > 0 {
		fmt.Println()
		fmt.Println("  Assumptions")
		for _, a := range b.Assumptions {
			fmt.Printf("    - %s\n", a)
		}
	}
}

func printBaselineJSON(b service.ComputedBaseline) error {
	type metricWire struct {
		Key         string   `json:"key"`
		Value       string   `json:"value"`
		Calculation string   `json:"calculation"`
		Confidence  string   `json:"confidence"`
		EvidenceIDs []string `json:"evidence_ids"`
		Assumptions []string `json:"assumptions,omitempty"`
	}
	metrics := make([]metricWire, len(b.Metrics))
	for i, m := range b.Metrics {
		metrics[i] = metricWire{
			Key:         m.Key,
			Value:       m.Value.StringFixed(2),
			Calculation: m.Calculation,
			Confidence:  m.Confidence,
			EvidenceIDs: m.EvidenceIDs,
			Assumptions: m.Assumptions,
		}
	}
	payload := map[string]any{
		"period_from":                b.PeriodFrom.Format("2006-01-02"),
		"period_to":                  b.PeriodTo.Format("2006-01-02"),
		"algorithm_version":          b.AlgorithmVersion,
		"status":                     b.Status,
		"confidence":                 b.Confidence,
		"regular_monthly_income":     b.RegularMonthlyIncome.StringFixed(2),
		"monthly_fixed_costs":        b.MonthlyFixedCosts.StringFixed(2),
		"monthly_irregular_costs":    b.MonthlyIrregularCosts.StringFixed(2),
		"avg_variable_spend":         b.AvgVariableSpend.StringFixed(2),
		"sustainable_free_cashflow":  b.SustainableFreeCashflow.StringFixed(2),
		"assumptions":                b.Assumptions,
		"metrics":                    metrics,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}
