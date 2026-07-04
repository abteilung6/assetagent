package main

import (
	"fmt"

	"github.com/abteilung6/assetagent/internal/config"
	"github.com/abteilung6/assetagent/internal/db"
	"github.com/spf13/cobra"
)

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "up",
			Short: "Apply all pending migrations",
			RunE:  runMigrate("up"),
		},
		&cobra.Command{
			Use:   "down",
			Short: "Roll back the last migration",
			RunE:  runMigrate("down"),
		},
		&cobra.Command{
			Use:   "status",
			Short: "Show migration status",
			RunE:  runMigrate("status"),
		},
	)

	return cmd
}

func runMigrate(direction string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if cfg.DatabaseURL == "" {
			return fmt.Errorf("DATABASE_URL is required")
		}
		return db.RunMigrations(cfg.DatabaseURL, direction)
	}
}
