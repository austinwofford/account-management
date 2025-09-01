package webserver

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/austinwofford/account-management/internal/database"
	"github.com/austinwofford/account-management/internal/webserver/accounts"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

type Config struct {
	Address      string
	CORSEnabled  bool
	DebugEnabled bool
	PostgresURL  string
}

func NewHTTPServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func NewRouter(cfg Config, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	//r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	// TODO: Determine if I should use timeouts on the router or on the server
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(slogMiddleware(logger))
	//r.Use(middleware.Logger) TODO: Maybe use chi's logging middleware?

	if cfg.CORSEnabled {
		r.Use(cors.Handler(cors.Options{
			// TODO: only allow the origin(s) we expect
			AllowedOrigins: []string{"*"},
			// TODO: Remove unexpected methods
			AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
			MaxAge:         300,
		}))
	}

	ctx := context.Background()

	db, err := database.NewDB(cfg.PostgresURL)
	if err != nil {
		logger.ErrorContext(ctx, "fatal error creating database client", "error", err)
		os.Exit(1)
	}

	// u up?
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Mount("/v1/accounts", accounts.NewHandler(accounts.HandlerDeps{
		DB: db,
	}))

	return r
}
