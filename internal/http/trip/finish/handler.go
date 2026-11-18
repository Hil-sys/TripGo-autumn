package finish

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Hil-sys/TripGo-autumn/internal/http/domainerror"
	"github.com/Hil-sys/TripGo-autumn/internal/http/trip"
)

type Handler struct {
	uc      usecaseInterface
	metrics metricsRecorder
	logger  *slog.Logger
}

func NewHandler(uc usecaseInterface, metrics metricsRecorder, logger *slog.Logger) *Handler {
	return &Handler{
		uc:      uc,
		metrics: metrics,
		logger:  logger,
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, tripID string) {
	completed, err := h.uc.Execute(r.Context(), tripID)
	if err != nil {
		status, prob := domainerror.Problem(err, r.URL.Path)
		if status >= http.StatusInternalServerError && h.logger != nil {
			h.logger.ErrorContext(r.Context(), "failed to finish trip", slog.Any("error", err))
		}
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(prob)
		return
	}

	// Фиксируем метрику завершения поездки через интерфейс
	if h.metrics != nil {
		h.metrics.TripCompleted(r.Context())
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(trip.ToAPI(completed))
}
