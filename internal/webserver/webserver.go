package webserver

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/austinwofford/account-management/docs"
	"github.com/austinwofford/account-management/internal/config"
	"github.com/austinwofford/account-management/internal/database"
	"github.com/austinwofford/account-management/internal/service/auth"
	"github.com/austinwofford/account-management/internal/webserver/accounts"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

func NewHTTPServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func NewRouter(cfg config.Config, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(slogMiddleware())
	//TODO: Maybe use chi's logging middleware instead of mine?
	//r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	ctx := context.Background()

	db, err := database.NewDB(cfg.PostgresURL)
	if err != nil {
		logger.ErrorContext(ctx, "fatal error creating database client", "error", err)
		os.Exit(1)
	}

	// healthcheck
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		err := db.HealthCheck(ctx)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		w.WriteHeader(http.StatusOK)
	})

	// docs
	r.Handle("/docs/*", http.StripPrefix("/docs/", docs.Handler))

	r.Mount("/v1/accounts", accounts.NewHandler(accounts.HandlerDeps{
		DB: db,
		AuthClient: auth.NewClient(auth.Config{
			JWTSecretKey:           cfg.JWTSecretKey,
			AccessTokenTTLMinutes:  cfg.AccessTokenTTLMinutes,
			RefreshTokenTTLMinutes: cfg.RefreshTokenTTLMinutes,
		}),
	}))

	return r
}
