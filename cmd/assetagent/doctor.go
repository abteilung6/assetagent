package main

import (
	"context"
	"errors"
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
		Short: "Check database and LLM provider connectivity",
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

			registry, err := newLLMRegistry(cfg)
			if err != nil {
				return err
			}

			for _, providerID := range registry.EnabledProviders() {
				provider, err := registry.Resolve(ctx, providerID, "")
				if err != nil {
					if errors.Is(err, llm.ErrOpenRouterNoKey) {
						slog.Warn("openrouter enabled but OPENROUTER_API_KEY not set, skipping ping")
						continue
					}
					return fmt.Errorf("llm provider %s: %w", providerID, err)
				}

				if err := provider.Ping(ctx); err != nil {
					return fmt.Errorf("llm provider %s ping failed: %w", providerID, err)
				}
				slog.Info("llm provider ok", "provider", providerID, "model", provider.Model())
			}

			return nil
		},
	}
}
