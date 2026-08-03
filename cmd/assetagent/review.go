package main

import (
	"context"
	"fmt"

	"github.com/abteilung6/assetagent/internal/service"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func newReviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review",
		Short: "Developer tools for Money Review (not an end-user path)",
	}
	cmd.AddCommand(newReviewCreateCmd())
	cmd.AddCommand(newReviewListCmd())
	cmd.AddCommand(newReviewConfirmCmd())
	return cmd
}

func newReviewCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Generate a Money Review from the current baseline",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pool, cleanup, err := openImportDB()
			if err != nil {
				return err
			}
			defer cleanup()

			item, err := service.NewMoneyReview(pool).Create(context.Background(), nil)
			if err != nil {
				return err
			}
			printMoneyReview(item)
			return nil
		},
	}
}

func newReviewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List Money Reviews",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pool, cleanup, err := openImportDB()
			if err != nil {
				return err
			}
			defer cleanup()

			items, err := service.NewMoneyReview(pool).List(context.Background(), 20)
			if err != nil {
				return err
			}
			if len(items) == 0 {
				fmt.Println("No money reviews")
				return nil
			}
			for _, item := range items {
				fmt.Printf("%s  %-18s  findings=%d  %s\n",
					item.ID, item.Status, len(item.Findings), item.Summary)
			}
			return nil
		},
	}
}

func newReviewConfirmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "confirm [id]",
		Short: "Confirm a Money Review",
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

			item, err := service.NewMoneyReview(pool).Confirm(context.Background(), id)
			if err != nil {
				return err
			}
			printMoneyReview(item)
			return nil
		},
	}
}

func printMoneyReview(m service.MoneyReview) {
	fmt.Println("MoneyReview")
	fmt.Printf("  ID:       %s\n", m.ID)
	fmt.Printf("  Baseline: %s\n", m.BaselineID)
	fmt.Printf("  Period:   %s → %s\n", m.PeriodFrom.Format("2006-01-02"), m.PeriodTo.Format("2006-01-02"))
	fmt.Printf("  Status:   %s\n", m.Status)
	fmt.Printf("  Summary:  %s\n", m.Summary)
	if len(m.Findings) == 0 {
		fmt.Println("  Findings: (none)")
		return
	}
	fmt.Println("  Findings:")
	for _, f := range m.Findings {
		amount := ""
		if f.Amount != nil {
			amount = "  " + f.Amount.StringFixed(2)
		}
		fmt.Printf("    - [%s]%s  %s\n", f.Type, amount, f.Title)
	}
}
