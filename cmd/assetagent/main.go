package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/abteilung6/assetagent/internal/config"
	"github.com/spf13/cobra"
)

var version = "dev"

func main() {
	root := &cobra.Command{
		Use:   "assetagent",
		Short: "AI native wealth management agent",
	}

	root.Flags().Bool("version", false, "Print version and exit")
	root.AddCommand(newMigrateCmd())
	root.AddCommand(newDoctorCmd())
	root.AddCommand(newImportCmd())
	root.AddCommand(newClassifyCmd())
	root.AddCommand(newServeCmd())
	root.RunE = func(cmd *cobra.Command, args []string) error {
		showVersion, _ := cmd.Flags().GetBool("version")
		if showVersion {
			fmt.Println(version)
			return nil
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		slog.SetDefault(newLogger(cfg))
		slog.Info("assetagent ready", "version", version, "log_level", cfg.LogLevel.String())

		return cmd.Help()
	}

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func newLogger(cfg config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	if cfg.LogFormat == "json" {
		return slog.New(slog.NewJSONHandler(os.Stderr, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stderr, opts))
}
