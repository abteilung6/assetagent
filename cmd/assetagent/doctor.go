package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/abteilung6/assetagent/internal/config"
	"github.com/abteilung6/assetagent/internal/db"
	"github.com/abteilung6/assetagent/internal/llm"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check database and Ollama connectivity",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			slog.SetDefault(newLogger(cfg))

			ctx := context.Background()

			if cfg.DatabaseURL != "" {
				pool, err := db.NewPool(ctx, cfg.DatabaseURL)
				if err != nil {
					return err
				}
				defer pool.Close()

				health := repository.NewHealth(pool)
				if err := health.Ping(ctx); err != nil {
					return fmt.Errorf("database ping failed: %w", err)
				}
				slog.Info("database ok")
			} else {
				slog.Warn("DATABASE_URL not set, skipping database check")
			}

			ollama := llm.NewOllama(cfg.OllamaBaseURL, cfg.OllamaModel)
			if err := ollama.Ping(ctx); err != nil {
				return fmt.Errorf("ollama check failed: %w", err)
			}
			slog.Info("ollama ok", "model", ollama.Model())

			return nil
		},
	}
}
