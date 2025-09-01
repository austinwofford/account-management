package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/austinwofford/account-management/internal/webserver"
)

func main() {
	// TODO: set logger default to log request IDs with every log output.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	//logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := webserver.Config{
		Address:     ":8080",
		CORSEnabled: true,
	}

	h := webserver.NewRouter(cfg, logger)
	srv := webserver.NewHTTPServer(cfg.Address, h)

	// start the webserver in a goroutine
	errCh := make(chan error, 1)

	go func() {
		logger.Info("starting webserver", "addr", cfg.Address)
		// TODO: serve TLS?
		errCh <- srv.ListenAndServe()
	}()

	// wait for signal or fatal listen error
	select {
	case sig := <-trap():
		logger.Info("shutdown signal", "signal", sig)
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "err", err)
		}
	}

	// graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "err", err)
	} else {
		logger.Info("server stopped")
	}
}

func trap() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	return ch
}
