package create

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	domain "github.com/Hil-sys/TripGo-autumn/internal/domain/trip"
	"github.com/Hil-sys/TripGo-autumn/internal/generated"
	"github.com/Hil-sys/TripGo-autumn/internal/http/domainerror"
	"github.com/Hil-sys/TripGo-autumn/internal/http/trip"
	"github.com/Hil-sys/TripGo-autumn/internal/model"
	usecase "github.com/Hil-sys/TripGo-autumn/internal/usecase/trip/create"
)

type rawCoordinates struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

type rawCreateTripRequest struct {
	UserID     *uuid.UUID      `json:"user_id"`
	DriverID   *uuid.UUID      `json:"driver_id"`
	StartPoint *rawCoordinates `json:"start_point"`
	EndPoint   *rawCoordinates `json:"end_point"`
	Price      *int64          `json:"price"`
}

const errNullOrMissing = "cannot be null or missing"

func (req *rawCreateTripRequest) validate() error {
	if req.UserID == nil || *req.UserID == uuid.Nil {
		return domain.ValidationError{Field: "user_id", Reason: "cannot be empty or null"}
	}
	if req.DriverID == nil || *req.DriverID == uuid.Nil {
		return domain.ValidationError{Field: "driver_id", Reason: "cannot be empty or null"}
	}
	if req.StartPoint == nil {
		return domain.ValidationError{Field: "start_point", Reason: errNullOrMissing}
	}
	if req.StartPoint.Latitude == nil {
		return domain.ValidationError{Field: "start_point.latitude", Reason: errNullOrMissing}
	}
	if req.StartPoint.Longitude == nil {
		return domain.ValidationError{Field: "start_point.longitude", Reason: errNullOrMissing}
	}
	if req.EndPoint == nil {
		return domain.ValidationError{Field: "end_point", Reason: errNullOrMissing}
	}
	if req.EndPoint.Latitude == nil {
		return domain.ValidationError{Field: "end_point.latitude", Reason: errNullOrMissing}
	}
	if req.EndPoint.Longitude == nil {
		return domain.ValidationError{Field: "end_point.longitude", Reason: errNullOrMissing}
	}
	if req.Price == nil {
		return domain.ValidationError{Field: "price", Reason: errNullOrMissing}
	}
	return nil
}

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

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req rawCreateTripRequest
	if err := dec.Decode(&req); err != nil {
		status, prob := domainerror.Problem(
			domain.ValidationError{Field: "body", Reason: fmt.Sprintf("invalid json payload: %v", err)},
			r.URL.Path,
		)
		writeProblem(w, status, prob)
		return
	}

	var extra json.RawMessage
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		status, prob := domainerror.Problem(
			domain.ValidationError{Field: "body", Reason: "unexpected data after json object"},
			r.URL.Path,
		)
		writeProblem(w, status, prob)
		return
	}

	if err := req.validate(); err != nil {
		status, prob := domainerror.Problem(err, r.URL.Path)
		writeProblem(w, status, prob)
		return
	}

	created, err := h.uc.Execute(r.Context(), usecase.Input{
		UserID:   req.UserID.String(),
		DriverID: req.DriverID.String(),
		StartPoint: model.Point{
			Latitude:  *req.StartPoint.Latitude,
			Longitude: *req.StartPoint.Longitude,
		},
		EndPoint: model.Point{
			Latitude:  *req.EndPoint.Latitude,
			Longitude: *req.EndPoint.Longitude,
		},
		Price: *req.Price,
	})
	if err != nil {
		status, prob := domainerror.Problem(err, r.URL.Path)
		if status >= http.StatusInternalServerError && h.logger != nil {
			h.logger.ErrorContext(r.Context(), "failed to create trip", slog.Any("error", err))
		}
		writeProblem(w, status, prob)
		return
	}

	// Фиксируем метрику создания поездки через интерфейс
	if h.metrics != nil {
		h.metrics.TripCreated(r.Context())
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
