package create

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	domain "github.com/Hil-sys/TripGo-autumn/internal/domain/trip"
	"github.com/Hil-sys/TripGo-autumn/internal/generated"
	"github.com/Hil-sys/TripGo-autumn/internal/http/domainerror"
	"github.com/Hil-sys/TripGo-autumn/internal/http/trip"
	"github.com/Hil-sys/TripGo-autumn/internal/infra/telemetry"
	"github.com/Hil-sys/TripGo-autumn/internal/model"
	usecase "github.com/Hil-sys/TripGo-autumn/internal/usecase/trip/create"
)

type usecaseInterface interface {
	Execute(ctx context.Context, input usecase.Input) (model.Trip, error)
}

type Handler struct {
	uc      usecaseInterface
	metrics *telemetry.Metrics
	logger  *slog.Logger
}

func NewHandler(uc usecaseInterface, metrics *telemetry.Metrics) *Handler {
	return &Handler{
		uc:      uc,
		metrics: metrics,
		logger:  slog.Default(),
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	var req api.CreateTripJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		status, prob := domainerror.Problem(
			domain.ValidationError{Field: "body", Reason: fmt.Sprintf("invalid json payload: %v", err)},
			r.URL.Path,
		)
		writeProblem(w, status, prob)
		return
	}

	created, err := h.uc.Execute(r.Context(), usecase.Input{
		UserID:   req.UserId.String(),
		DriverID: req.DriverId.String(),
		StartPoint: model.Point{
			Latitude:  req.StartPoint.Latitude,
			Longitude: req.StartPoint.Longitude,
		},
		EndPoint: model.Point{
			Latitude:  req.EndPoint.Latitude,
			Longitude: req.EndPoint.Longitude,
		},
		Price: req.Price,
	})
	if err != nil {
		status, prob := domainerror.Problem(err, r.URL.Path)
		if status >= http.StatusInternalServerError {
			h.logger.ErrorContext(r.Context(), "failed to create trip", slog.Any("error", err))
		}
		writeProblem(w, status, prob)
		return
	}

	// Фиксируем метрику tripgo.trips.created
	if h.metrics != nil && h.metrics.TripsCreated != nil {
		h.metrics.TripsCreated.Add(r.Context(), 1)
	}

	w.Header().Set("Location", fmt.Sprintf("/api/v1/trips/%s", created.ID))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(trip.ToAPI(created))
}

func writeProblem(w http.ResponseWriter, status int, prob api.Problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(prob)
}
