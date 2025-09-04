package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/austinwofford/account-management/internal/config"
	"github.com/austinwofford/account-management/internal/webserver"
)

func main() {
	// TODO: set logger default to log request IDs with every log output.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("fatal error loading config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	router, err := webserver.NewRouter(*cfg, logger)
	if err != nil {
		logger.ErrorContext(ctx, "fatal error creating database client", "error", err)
		os.Exit(1)
	}

	srv := webserver.NewHTTPServer(cfg.HTTPAddress, router)

	// err chan for server errors
	errCh := make(chan error, 1)

	// start the webserver in a go routine and listen for errors
	go func() {
		logger.InfoContext(ctx, "starting webserver", "addr", cfg.HTTPAddress)
		errCh <- srv.ListenAndServe()
	}()

	// wait for signal or fatal listen error
	select {
	case sig := <-trap():
		// if we get a shutdown signal, all is good, let's gracefully shutdown
		logger.InfoContext(ctx, "shutdown signal received", "signal", sig)
	case err := <-errCh:
		// if we get an error from the Listen and Serve, log it and exit
		if err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	}

	// graceful shutdown
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "err", err)
	} else {
		logger.Info("server stopped")
	}
}

// trap returns a channel that receives OS shutdown signals
// so that we may gracefully shutdown
func trap() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	return ch
}
