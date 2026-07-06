package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/config"
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

			slog.SetDefault(newLogger(cfg))

			router := chi.NewRouter()
			gen.HandlerFromMux(handler.New(), router)

			slog.Info("api listening", "addr", cfg.APIAddr)
			if err := http.ListenAndServe(cfg.APIAddr, router); err != nil {
				return fmt.Errorf("serve: %w", err)
			}

			return nil
		},
	}
}
