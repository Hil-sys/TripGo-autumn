package get

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Hil-sys/TripGo-autumn/internal/http/domainerror"
	"github.com/Hil-sys/TripGo-autumn/internal/http/trip"
	"github.com/Hil-sys/TripGo-autumn/internal/model"
)

type usecaseInterface interface {
	Execute(ctx context.Context, id string) (model.Trip, error)
}

type Handler struct {
	uc     usecaseInterface
	logger *slog.Logger
}

func NewHandler(uc usecaseInterface, logger *slog.Logger) *Handler {
	return &Handler{
		uc:     uc,
		logger: logger,
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, tripID string) {
	foundTrip, err := h.uc.Execute(r.Context(), tripID)
	if err != nil {
		status, prob := domainerror.Problem(err, r.URL.Path)
		if status >= http.StatusInternalServerError && h.logger != nil {
			h.logger.ErrorContext(r.Context(), "failed to get trip", slog.Any("error", err))
		}
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(prob)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(trip.ToAPI(foundTrip))
}
