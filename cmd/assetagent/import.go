package main

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/abteilung6/assetagent/internal/config"
	"github.com/abteilung6/assetagent/internal/db"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	var dryRun bool
	var accountName string

	cmd := &cobra.Command{
		Use:   "import [file.csv]",
		Short: "Import Sparkasse CSV transactions",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			path := args[0]

			if dryRun {
				preview, err := service.PreviewFile(path)
				if err != nil {
					return err
				}
				printImportPreview(path, preview)
				return nil
			}

			pool, cleanup, err := openImportDB()
			if err != nil {
				return err
			}
			defer cleanup()

			importer := service.NewImport(pool)
			result, err := importer.ImportFile(context.Background(), path, domain.ImportOptions{
				AccountName: accountName,
			})
			if err != nil {
				return err
			}

			printImportResult(path, result)
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Parse and validate the CSV without writing to the database")
	cmd.Flags().StringVar(&accountName, "account-name", "", "Display name for the account (created on first import)")

	cmd.AddCommand(newImportRunsCmd())
	cmd.AddCommand(newImportRollbackCmd())
	return cmd
}

func newImportRunsCmd() *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "runs",
		Short: "List recent import runs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			pool, cleanup, err := openImportDB()
			if err != nil {
				return err
			}
			defer cleanup()

			runs, err := service.NewImport(pool).ListRuns(context.Background(), limit)
			if err != nil {
				return err
			}
			printImportRuns(runs)
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of runs to list")
	return cmd
}

func newImportRollbackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rollback [run-id]",
		Short: "Undo a committed import run and delete its transactions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runID, err := uuid.Parse(args[0])
			if err != nil {
				return fmt.Errorf("invalid run id: %w", err)
			}

			pool, cleanup, err := openImportDB()
			if err != nil {
				return err
			}
			defer cleanup()

			result, err := service.NewImport(pool).Rollback(context.Background(), runID)
			if err != nil {
				return err
			}
			printImportRollback(result)
			return nil
		},
	}
}

func openImportDB() (*pgxpool.Pool, func(), error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}
	if cfg.DatabaseURL == "" {
		return nil, nil, fmt.Errorf("DATABASE_URL is required")
	}

	slog.SetDefault(newLogger(cfg))

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	return pool, func() { pool.Close() }, nil
}

func printImportResult(path string, result domain.ImportResult) {
	fmt.Println("Import complete")
	fmt.Printf("  File:       %s\n", filepath.Base(path))
	fmt.Printf("  Account:    %s (%s)\n", result.AccountName, result.AccountID)
	fmt.Printf("  Import run: %s\n", result.ImportRunID)
	fmt.Printf("  Rows:       %d\n", result.Rows)
	fmt.Printf("  Inserted:   %d\n", result.Inserted)
	fmt.Printf("  Duplicates: %d\n", result.Duplicates)
	fmt.Printf("  Errors:     %d\n", result.Errors)
}

func printImportRuns(runs []domain.ImportRunSummary) {
	if len(runs) == 0 {
		fmt.Println("No import runs found")
		return
	}

	fmt.Println("Import runs")
	for _, run := range runs {
		created := run.CreatedAt.Format("2006-01-02 15:04:05")
		fmt.Printf("  %s  %-12s  inserted=%-4d dup=%-4d  %s  %s\n",
			run.ID,
			run.Status,
			run.RowInserted,
			run.RowDuplicate,
			created,
			run.SourceFilename,
		)
	}
}

func printImportRollback(result domain.ImportRollbackResult) {
	fmt.Println("Import rollback complete")
	fmt.Printf("  Import run: %s\n", result.ImportRunID)
	fmt.Printf("  File:       %s\n", result.SourceFilename)
	fmt.Printf("  Deleted:    %d transactions\n", result.Deleted)
}

func printImportPreview(path string, preview domain.ImportPreview) {
	fmt.Println("Import preview (dry-run — no database writes)")
	fmt.Printf("  File:              %s\n", filepath.Base(path))
	fmt.Printf("  Parser:            %s %s\n", preview.ParserName, preview.ParserVersion)
	fmt.Printf("  File hash:         %s\n", preview.FileHash)
	if preview.SuggestedAccount != "" {
		fmt.Printf("  Suggested account: %s\n", preview.SuggestedAccount)
	}
	if preview.PeriodFrom != nil && preview.PeriodTo != nil {
		fmt.Printf("  Period:            %s → %s\n",
			preview.PeriodFrom.Format("2006-01-02"),
			preview.PeriodTo.Format("2006-01-02"),
		)
	} else {
		fmt.Printf("  Period:            (none)\n")
	}
	fmt.Printf("  Rows total:        %d\n", preview.RowTotal)
	fmt.Printf("  Rows valid:        %d\n", preview.RowValid)
	fmt.Printf("  Rows invalid:      %d\n", preview.RowInvalid)

	if len(preview.Warnings) > 0 {
		fmt.Println("  Warnings:")
		for _, warning := range preview.Warnings {
			fmt.Printf("    - %s\n", warning)
		}
	}

	if len(preview.SampleRows) > 0 {
		fmt.Println("  Sample rows:")
		for _, row := range preview.SampleRows {
			purpose := row.Purpose
			if purpose == "" {
				purpose = "—"
			}
			counterparty := row.Counterparty
			if counterparty == "" {
				counterparty = "—"
			}
			fmt.Printf("    %s  %10s %s  %s  %s\n",
				row.BookingDate.Format("2006-01-02"),
				row.Amount,
				row.Currency,
				truncateCLI(counterparty, 28),
				truncateCLI(purpose, 40),
			)
		}
	}

	if len(preview.InvalidRows) > 0 {
		fmt.Println("  Invalid rows:")
		for _, row := range preview.InvalidRows {
			field := row.Field
			if field == "" {
				field = "row"
			}
			fmt.Printf("    line %d [%s]: %s\n", row.Line, field, row.Message)
		}
		if preview.RowInvalid > len(preview.InvalidRows) {
			fmt.Printf("    … %d more\n", preview.RowInvalid-len(preview.InvalidRows))
		}
	}
}

func truncateCLI(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max] + "…"
}
