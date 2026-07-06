package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/config"
	"github.com/abteilung6/assetagent/internal/db"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP API server",
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

			txRepo := repository.NewTransaction(pool)
			listSvc := service.NewList(txRepo)

			router := chi.NewRouter()
			gen.HandlerWithOptions(handler.New(listSvc), gen.ChiServerOptions{
				BaseRouter:       router,
				ErrorHandlerFunc: handler.APIErrorHandler,
			})

			slog.Info("api listening", "addr", cfg.APIAddr)
			if err := http.ListenAndServe(cfg.APIAddr, router); err != nil {
				return fmt.Errorf("serve: %w", err)
			}

			return nil
		},
	}
}
