package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
	bytesCount int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriterWrapper) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesCount += n
	return n, err
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

			next.ServeHTTP(ww, r)

			duration := time.Since(start)

			attrs := []any{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.statusCode),
				slog.Duration("duration", duration),
				slog.Int("bytes", ww.bytesCount),
				slog.String("remote_addr", r.RemoteAddr),
			}

			if ww.statusCode >= 500 {
				logger.ErrorContext(r.Context(), "http request completed with server error", attrs...)
			} else {
				logger.InfoContext(r.Context(), "http request completed", attrs...)
			}
		})
	}
}
