package finish

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Hil-sys/TripGo-autumn/internal/http/domainerror"
	"github.com/Hil-sys/TripGo-autumn/internal/http/trip"
	"github.com/Hil-sys/TripGo-autumn/internal/infra/telemetry"
	"github.com/Hil-sys/TripGo-autumn/internal/model"
)

type usecaseInterface interface {
	Execute(ctx context.Context, id string) (model.Trip, error)
}

type Handler struct {
	uc      usecaseInterface
	metrics *telemetry.Metrics
}

func NewHandler(uc usecaseInterface, metrics *telemetry.Metrics) *Handler {
	return &Handler{
		uc:      uc,
		metrics: metrics,
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, tripID string) {
	completed, err := h.uc.Execute(r.Context(), tripID)
	if err != nil {
		status, prob := domainerror.Problem(err, r.URL.Path)
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(prob)
		return
	}

	// Фиксируем метрику tripgo.trips.completed
	if h.metrics != nil && h.metrics.TripsCompleted != nil {
		h.metrics.TripsCompleted.Add(r.Context(), 1)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(trip.ToAPI(completed))
}
