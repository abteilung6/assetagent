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
