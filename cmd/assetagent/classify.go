package main

import (
	"context"
	"fmt"

	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func newClassifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "classify",
		Short: "Developer tools for money classification (not an end-user path)",
	}
	cmd.AddCommand(newClassifyCategoriesCmd())
	cmd.AddCommand(newClassifyTransfersCmd())
	cmd.AddCommand(newClassifyMerchantsCmd())
	cmd.AddCommand(newClassifyRecurringCmd())
	cmd.AddCommand(newClassifyRunCmd())
	cmd.AddCommand(newClassifyCorrectCmd())
	return cmd
}

func newClassifyCategoriesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "categories",
		Short: "List system category taxonomy",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pool, cleanup, err := openImportDB()
			if err != nil {
				return err
			}
			defer cleanup()

			categories, err := repository.NewCategories(pool).List(context.Background())
			if err != nil {
				return err
			}

			if len(categories) == 0 {
				fmt.Println("No categories found (run migrate up)")
				return nil
			}

			fmt.Println("Categories")
			for _, cat := range categories {
				system := ""
				if cat.IsSystem {
					system = " system"
				}
				fmt.Printf("  %-18s  %-20s  kind=%-8s%s\n",
					cat.Slug,
					cat.DisplayName,
					cat.Kind,
					system,
				)
			}
			return nil
		},
	}
}

func newClassifyTransfersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfers",
		Short: "Internal transfer pairing (developer)",
	}
	cmd.AddCommand(newClassifyTransfersScanCmd())
	cmd.AddCommand(newClassifyTransfersListCmd())
	cmd.AddCommand(newClassifyTransfersConfirmCmd())
	cmd.AddCommand(newClassifyTransfersRejectCmd())
	return cmd
}

func newClassifyTransfersScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Detect and store suggested internal transfer pairs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pool, cleanup, err := openImportDB()
			if err != nil {
				return err
			}
			defer cleanup()

			result, err := service.NewTransfers(pool).Scan(context.Background())
			if err != nil {
				return err
			}

			fmt.Println("Transfer scan complete")
			fmt.Printf("  Transactions considered: %d\n", result.CandidatesConsidered)
			fmt.Printf("  Already paired legs:     %d\n", result.SkippedExisting)
			fmt.Printf("  New suggestions:         %d\n", result.Suggested)
			for _, pair := range result.Pairs {
				fmt.Printf("  %s  %-9s  out=%s  in=%s\n",
					pair.ID, pair.Confidence, pair.TxOutID, pair.TxInID)
			}
			return nil
		},
	}
}

func newClassifyTransfersListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List stored transfer pairs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pool, cleanup, err := openImportDB()
			if err != nil {
				return err
			}
			defer cleanup()

			pairs, err := service.NewTransfers(pool).List(context.Background())
			if err != nil {
				return err
			}
			if len(pairs) == 0 {
				fmt.Println("No transfer pairs found")
				return nil
			}
			fmt.Println("Transfer pairs")
			for _, pair := range pairs {
				fmt.Printf("  %s  %-10s  %-9s  out=%s  in=%s\n",
					pair.ID, pair.Status, pair.Confidence, pair.TxOutID, pair.TxInID)
			}
			return nil
		},
	}
}

func newClassifyTransfersConfirmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "confirm <pair-id>",
		Short: "Confirm a suggested transfer pair (excludes legs from cashflow v2)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return decideTransferPair(args[0], true)
		},
	}
}

func newClassifyTransfersRejectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reject <pair-id>",
		Short: "Reject a suggested transfer pair",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return decideTransferPair(args[0], false)
		},
	}
}

func decideTransferPair(idArg string, confirm bool) error {
	id, err := uuid.Parse(idArg)
	if err != nil {
		return fmt.Errorf("invalid pair id: %w", err)
	}

	pool, cleanup, err := openImportDB()
	if err != nil {
		return err
	}
	defer cleanup()

	svc := service.NewTransfers(pool)
	var pair domain.TransferPair
	if confirm {
		pair, err = svc.Confirm(context.Background(), id)
	} else {
		pair, err = svc.Reject(context.Background(), id)
	}
	if err != nil {
		return err
	}

	action := "rejected"
	if confirm {
		action = "confirmed"
	}
	fmt.Printf("Transfer pair %s\n", action)
	fmt.Printf("  ID:         %s\n", pair.ID)
	fmt.Printf("  Status:     %s\n", pair.Status)
	fmt.Printf("  Confidence: %s\n", pair.Confidence)
	fmt.Printf("  Out tx:     %s\n", pair.TxOutID)
	fmt.Printf("  In tx:      %s\n", pair.TxInID)
	return nil
}

func newClassifyMerchantsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "merchants",
		Short: "Merchant normalization (developer)",
	}
	cmd.AddCommand(newClassifyMerchantsRebuildCmd())
	cmd.AddCommand(newClassifyMerchantsListCmd())
	return cmd
}

func newClassifyRecurringCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recurring",
		Short: "Recurring payment series detection (developer)",
	}
	cmd.AddCommand(newClassifyRecurringScanCmd())
	cmd.AddCommand(newClassifyRecurringListCmd())
	return cmd
}

func newClassifyRecurringScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan",
		Short: "Detect and store recurring payment series",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pool, cleanup, err := openImportDB()
			if err != nil {
				return err
			}
			defer cleanup()

			result, err := service.NewRecurring(pool).Scan(context.Background())
			if err != nil {
				return err
			}

			fmt.Println("Recurring scan complete")
			fmt.Printf("  Transactions considered: %d\n", result.TransactionsConsidered)
			fmt.Printf("  Already assigned / skip: %d\n", result.SkippedExisting)
			fmt.Printf("  New series:             %d\n", result.Suggested)
			for _, series := range result.Series {
				changed := ""
				if series.AmountChanged {
					changed = " amount_changed"
				}
				fmt.Printf("  %s  %-8s  %-16s  %s  members=%d  %s%s\n",
					series.ID,
					series.Interval,
					series.Kind,
					series.Status,
					series.MemberCount,
					series.DisplayName,
					changed,
				)
			}
			return nil
		},
	}
}

func newClassifyRecurringListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List stored recurring series",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pool, cleanup, err := openImportDB()
			if err != nil {
				return err
			}
			defer cleanup()

			series, err := service.NewRecurring(pool).List(context.Background())
			if err != nil {
				return err
			}
			if len(series) == 0 {
				fmt.Println("No recurring series found (run classify recurring scan)")
				return nil
			}
			fmt.Println("Recurring series")
			for _, s := range series {
				changed := ""
				if s.AmountChanged {
					changed = " amount_changed"
				}
				fmt.Printf("  %s  %-8s  %-10s  %-16s  typical=%s  last=%s  members=%d  %s%s\n",
					s.ID,
					s.Interval,
					s.Status,
					s.Kind,
					s.AmountTypical.StringFixed(2),
					s.AmountLast.StringFixed(2),
					s.MemberCount,
					s.DisplayName,
					changed,
				)
			}
			return nil
		},
	}
}

func newClassifyMerchantsRebuildCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rebuild",
		Short: "Build merchants and aliases from transaction counterparties",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pool, cleanup, err := openImportDB()
			if err != nil {
				return err
			}
			defer cleanup()

			result, err := service.NewMerchants(pool).Rebuild(context.Background())
			if err != nil {
				return err
			}
			fmt.Println("Merchant rebuild complete")
			fmt.Printf("  Labels considered: %d\n", result.LabelsConsidered)
			fmt.Printf("  Merchants created: %d\n", result.MerchantsCreated)
			fmt.Printf("  Aliases created:   %d\n", result.AliasesCreated)
			fmt.Printf("  Aliases existing:  %d\n", result.AliasesExisting)
			fmt.Printf("  Skipped empty:     %d\n", result.SkippedEmpty)
			return nil
		},
	}
}

func newClassifyMerchantsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List normalized merchants",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pool, cleanup, err := openImportDB()
			if err != nil {
				return err
			}
			defer cleanup()

			merchants, err := service.NewMerchants(pool).List(context.Background())
			if err != nil {
				return err
			}
			if len(merchants) == 0 {
				fmt.Println("No merchants found (run classify merchants rebuild)")
				return nil
			}
			fmt.Println("Merchants")
			for _, m := range merchants {
				fmt.Printf("  %-28s  aliases=%d  %s\n", m.DisplayName, m.AliasCount, m.ID)
			}
			return nil
		},
	}
}

func newClassifyRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Classify all transactions into categories",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pool, cleanup, err := openImportDB()
			if err != nil {
				return err
			}
			defer cleanup()

			result, err := service.NewClassify(pool).Run(context.Background())
			if err != nil {
				return err
			}
			fmt.Println("Classify run complete")
			fmt.Printf("  Transactions:   %d\n", result.Transactions)
			fmt.Printf("  Upserted:       %d\n", result.Upserted)
			fmt.Printf("  Skipped (user): %d\n", result.SkippedUser)
			if len(result.BySource) > 0 {
				fmt.Println("  By source:")
				for source, count := range result.BySource {
					fmt.Printf("    %-12s %d\n", source, count)
				}
			}
			if len(result.ByCategory) > 0 {
				fmt.Println("  By category:")
				for slug, count := range result.ByCategory {
					fmt.Printf("    %-18s %d\n", slug, count)
				}
			}
			return nil
		},
	}
}

func newClassifyCorrectCmd() *cobra.Command {
	var category string
	var applyToMerchant bool

	cmd := &cobra.Command{
		Use:   "correct <transaction-id>",
		Short: "Set a transaction category and optionally remember it for the merchant",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if category == "" {
				return fmt.Errorf("--category is required")
			}
			txID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid transaction id: %w", err)
			}

			pool, cleanup, err := openImportDB()
			if err != nil {
				return err
			}
			defer cleanup()

			result, err := service.NewClassify(pool).Correct(context.Background(), txID, domain.ClassifyCorrectOptions{
				CategorySlug:    category,
				ApplyToMerchant: applyToMerchant,
			})
			if err != nil {
				return err
			}

			fmt.Println("Classification corrected")
			fmt.Printf("  Transaction: %s\n", result.TransactionID)
			fmt.Printf("  Category:    %s\n", result.CategorySlug)
			if result.RuleCreated && result.MerchantID != nil {
				fmt.Printf("  Merchant rule saved for %s\n", *result.MerchantID)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&category, "category", "", "Category slug (e.g. groceries, leisure)")
	cmd.Flags().BoolVar(&applyToMerchant, "apply-to-merchant", false, "Remember this category for the merchant on future classify runs")
	return cmd
}
