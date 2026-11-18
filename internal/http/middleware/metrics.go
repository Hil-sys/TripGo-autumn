package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"
)

type MetricsRecorder interface {
	RecordHTTPRequest(ctx context.Context, duration float64, method, route string, statusCode int)
	IncActiveRequests(ctx context.Context, method, route string)
	DecActiveRequests(ctx context.Context, method, route string)
}

func Metrics(m MetricsRecorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Пропускаем healthcheck эндпоинты
			if r.URL.Path == "/health" || r.URL.Path == "/ready" {
				next.ServeHTTP(w, r)
				return
			}

			route := matchRoute(r.URL.Path)
			method := r.Method

			m.IncActiveRequests(r.Context(), method, route)
			defer m.DecActiveRequests(r.Context(), method, route)

			start := time.Now()
			ww := &responseWriterWrapper{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			panicked := true
			defer func() {
				duration := time.Since(start).Seconds()
				statusCode := ww.statusCode
				if panicked {
					statusCode = http.StatusInternalServerError
				}
				m.RecordHTTPRequest(r.Context(), duration, method, route, statusCode)
			}()

			next.ServeHTTP(ww, r)
			panicked = false
		})
	}
}

// matchRoute возвращает фиксированный шаблон маршрута или "unmatched" для неизвестных путей,
// предотвращая разрастание cardinality временных рядов в Prometheus.
func matchRoute(path string) string {
	if path == "/api/v1/trips" {
		return "/api/v1/trips"
	}
	if strings.HasPrefix(path, "/api/v1/trips/") {
		rest := strings.TrimPrefix(path, "/api/v1/trips/")
		if rest == "" {
			return "unmatched"
		}
		parts := strings.Split(rest, "/")
		if len(parts) == 1 && parts[0] != "" {
			return "/api/v1/trips/{tripId}"
		}
		if len(parts) == 2 && parts[1] == "finish" {
			return "/api/v1/trips/{tripId}/finish"
		}
	}
	return "unmatched"
}
