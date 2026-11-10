package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/Hil-sys/TripGo-autumn/internal/infra/telemetry"
)

func Metrics(m *telemetry.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m == nil || m.HTTPDuration == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Пропускаем healthcheck эндпоинты
			if r.URL.Path == "/health" || r.URL.Path == "/ready" {
				next.ServeHTTP(w, r)
				return
			}

			route := getRoutePattern(r)
			method := r.Method

			activeAttrs := []attribute.KeyValue{
				attribute.String("http.request.method", method),
				attribute.String("http.route", route),
			}

			// Увеличиваем счетчик активных запросов
			m.HTTPActiveRequests.Add(r.Context(), 1, metric.WithAttributes(activeAttrs...))
			defer m.HTTPActiveRequests.Add(r.Context(), -1, metric.WithAttributes(activeAttrs...))

			start := time.Now()
			ww := &responseWriterWrapper{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(ww, r)

			duration := time.Since(start).Seconds()

			// Повторно сверяем маршрут после работы роутера Chi
			if p := getRoutePattern(r); p != "" {
				route = p
			}

			m.HTTPDuration.Record(r.Context(), duration, metric.WithAttributes(
				attribute.String("http.request.method", method),
				attribute.String("http.route", route),
				attribute.Int("http.response.status_code", ww.statusCode),
			))
		})
	}
}

// getRoutePattern возвращает шаблон маршрута без подстановки конкретных UUID
func getRoutePattern(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if pattern := rctx.RoutePattern(); pattern != "" {
			return pattern
		}
	}

	path := r.URL.Path
	if strings.HasPrefix(path, "/api/v1/trips/") {
		if strings.HasSuffix(path, "/finish") {
			return "/api/v1/trips/{tripId}/finish"
		}
		return "/api/v1/trips/{tripId}"
	}
	return path
}
