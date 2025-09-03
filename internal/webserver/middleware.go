package webserver

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/middleware"
)

// slogMiddleware logs http requests using slog
func slogMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &wrapWriter{ResponseWriter: w, code: http.StatusOK}
			next.ServeHTTP(ww, r)
			slog.Info("http_request",
				slog.String("http_method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status_code", ww.code),
				slog.Duration("duration", time.Since(start)),
				slog.String("remote_addr", r.RemoteAddr),
				slog.Any("request_id", r.Context().Value(middleware.RequestIDKey)),
			)
		})
	}
}

type wrapWriter struct {
	http.ResponseWriter
	code int
}

func (w *wrapWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}
