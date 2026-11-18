package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" || r.URL.Path == "/ready" {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			ww := &responseWriterWrapper{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			route := matchRoute(r.URL.Path)
			panicked := true
			defer func() {
				duration := time.Since(start)
				statusCode := ww.statusCode
				if panicked {
					statusCode = http.StatusInternalServerError
				}

				attrs := []any{
					slog.String("method", r.Method),
					slog.String("route", route),
					slog.Int("status", statusCode),
					slog.Int64("duration_ms", duration.Milliseconds()),
				}

				if statusCode >= http.StatusInternalServerError {
					logger.ErrorContext(r.Context(), "request handled", attrs...)
				} else {
					logger.InfoContext(r.Context(), "request handled", attrs...)
				}
			}()

			next.ServeHTTP(ww, r)
			panicked = false
		})
	}
}
