package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/abteilung6/assetagent/internal/api/gen"
	"github.com/abteilung6/assetagent/internal/api/handler"
	"github.com/abteilung6/assetagent/internal/api/middleware"
	"github.com/abteilung6/assetagent/internal/chat"
	"github.com/abteilung6/assetagent/internal/chat/tools"
	"github.com/abteilung6/assetagent/internal/config"
	"github.com/abteilung6/assetagent/internal/db"
	"github.com/abteilung6/assetagent/internal/repository"
	"github.com/abteilung6/assetagent/internal/service"
	"github.com/abteilung6/assetagent/internal/telemetry"
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
			shutdownTelemetry, err := telemetry.Init(ctx, telemetry.Config{
				Enabled:      cfg.LangfuseEnabled,
				PublicKey:    cfg.LangfusePublicKey,
				SecretKey:    cfg.LangfuseSecretKey,
				OTLPEndpoint: cfg.OTLPEndpoint,
				TraceDetail:  telemetry.ParseTraceDetail(cfg.LangfuseTraceDetail),
				ServiceName:  "assetagent",
			})
			if err != nil {
				return err
			}
			defer func() {
				_ = shutdownTelemetry(context.Background())
			}()

			pool, err := db.NewPool(ctx, cfg.DatabaseURL)
			if err != nil {
				return err
			}
			defer pool.Close()

			txRepo := repository.NewTransaction(pool)
			listSvc := service.NewList(txRepo)
			reportsRepo := repository.NewReports(pool)
			baselineSvc := service.NewBaseline(pool)
			moneyReviewSvc := service.NewMoneyReview(pool)
			forecastSvc := service.NewForecast(pool)
			classifySvc := service.NewClassify(pool)
			transfers := service.NewTransfers(pool)
			recurringSvc := service.NewRecurring(pool)
			toolRegistry := tools.NewRegistry(tools.Dependencies{
				Reports:     reportsRepo,
				Lister:      txRepo,
				Recurring:   recurringSvc,
				Baseline:    baselineSvc,
				Insights:    baselineSvc,
				MoneyReview: moneyReviewSvc,
				Forecast:    forecastSvc,
				Classify:    classifySvc,
				Transfers:   transfers,
				Queue:       classifySvc,
			})

			llmRegistry, err := newLLMRegistry(cfg)
			if err != nil {
				return err
			}
			chatCfg := chat.DefaultConfig()
			chatCfg.TraceDetail = telemetry.ParseTraceDetail(cfg.LangfuseTraceDetail)
			chatSvc := chat.NewRegistryService(
				llmRegistry,
				toolRegistry,
				chatCfg,
			)

			authRepo := repository.NewAuth(pool)
			sessions := service.NewSession(authRepo, service.SessionConfigFromEnv(
				cfg.AppEnv,
				cfg.SessionCookieName,
				cfg.SessionCookieSecure,
				cfg.SessionIdleHours,
				cfg.SessionAbsoluteHours,
			))

			var googleAuth *service.GoogleAuthService
			googleConfigured := cfg.GoogleClientID != "" && cfg.GoogleClientSecret != ""
			if googleConfigured {
				verifier, verr := service.NewRealGoogleIDTokenVerifier(ctx, cfg.GoogleClientID)
				if verr != nil {
					slog.Warn("google oidc verifier unavailable", "err", verr)
				} else {
					googleAuth = service.NewGoogleAuth(
						authRepo,
						sessions,
						service.NewRealGoogleOAuth(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL),
						verifier,
						service.GoogleAuthConfig{
							ClientID:          cfg.GoogleClientID,
							ClientSecret:      cfg.GoogleClientSecret,
							RedirectURL:       cfg.GoogleRedirectURL,
							FrontendURL:       cfg.FrontendURL,
							ClaimExistingData: cfg.AuthClaimExistingData,
							Configured:        true,
						},
					)
				}
			} else {
				slog.Info("google sign-in disabled (GOOGLE_CLIENT_ID/SECRET not set)")
			}

			router := chi.NewRouter()
			router.Use(middleware.CORSMiddleware(cfg.CORSAllowedOrigins))
			router.Use(middleware.ResolveSessionMiddleware(sessions))
			router.Use(middleware.RequireAuthMiddleware)

			if googleAuth != nil {
				gh := handler.NewGoogleAuthHandler(googleAuth, sessions)
				router.Get("/auth/google/start", gh.Start)
				router.Get("/auth/google/callback", gh.Callback)
			} else {
				unavailable := handler.NewGoogleAuthHandler(nil, sessions)
				router.Get("/auth/google/start", unavailable.Start)
				router.Get("/auth/google/callback", unavailable.Callback)
			}

			importer := service.NewImport(pool)
			categories := repository.NewCategories(pool)
			gen.HandlerWithOptions(handler.New(listSvc, chatSvc, llmRegistry, importer, transfers, classifySvc, categories, recurringSvc, baselineSvc, moneyReviewSvc, forecastSvc, service.NewDecision(pool), sessions), gen.ChiServerOptions{
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
