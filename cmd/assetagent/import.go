package main

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/abteilung6/assetagent/internal/config"
	"github.com/abteilung6/assetagent/internal/db"
	"github.com/abteilung6/assetagent/internal/domain"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import [file.csv]",
		Short: "Import Sparkasse CSV transactions",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if cfg.DatabaseURL == "" {
				return fmt.Errorf("DATABASE_URL is required")
			}

			slog.SetDefault(newLogger(cfg))

			ctx := context.Background()
			pool, err := db.NewPool(ctx, cfg.DatabaseURL)
			if err != nil {
				return err
			}
			defer pool.Close()

			importer := service.NewImport(repository.NewTransaction(pool))
			result, err := importer.ImportFile(ctx, args[0])
			if err != nil {
				return err
			}

			printImportResult(args[0], result)
			return nil
		},
	}
}

func printImportResult(path string, result domain.ImportResult) {
	fmt.Println("Import complete")
	fmt.Printf("  File:       %s\n", filepath.Base(path))
	fmt.Printf("  Rows:       %d\n", result.Rows)
	fmt.Printf("  Inserted:   %d\n", result.Inserted)
	fmt.Printf("  Duplicates: %d\n", result.Duplicates)
	fmt.Printf("  Errors:     %d\n", result.Errors)
}
