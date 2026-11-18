package middleware

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/Hil-sys/TripGo-autumn/internal/generated"
)

func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					err, ok := rec.(error)
					if !ok {
						err = fmt.Errorf("%v", rec)
					}

					stack := string(debug.Stack())
					logger.ErrorContext(r.Context(), "http request panicked",
						slog.Any("error", err),
						slog.String("stack", stack),
						slog.String("path", r.URL.Path),
						slog.String("method", r.Method),
					)

					w.Header().Set("Content-Type", "application/problem+json")
					w.WriteHeader(http.StatusInternalServerError)

					code := "internal_error"
					instance := r.URL.Path
					detail := "An unexpected error occurred"
					prob := api.Problem{
						Type:     "https://tripgo.example/problems/internal-error",
						Title:    "Internal Server Error",
						Status:   int32(http.StatusInternalServerError),
						Detail:   &detail,
						Code:     code,
						Instance: &instance,
					}
					_ = json.NewEncoder(w).Encode(prob)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
