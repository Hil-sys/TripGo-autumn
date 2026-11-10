package create

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Hil-sys/TripGo-autumn/internal/generated"
	"github.com/Hil-sys/TripGo-autumn/internal/http/domainerror"
	"github.com/Hil-sys/TripGo-autumn/internal/http/trip"
	"github.com/Hil-sys/TripGo-autumn/internal/model"
	usecase "github.com/Hil-sys/TripGo-autumn/internal/usecase/trip/create"
)

type usecaseInterface interface {
	Execute(ctx context.Context, input usecase.Input) (model.Trip, error)
}

type Handler struct {
	uc usecaseInterface
}

func NewHandler(uc usecaseInterface) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	var req api.CreateTripJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, r, domainerror.Problem(fmt.Errorf("invalid json body: %w", err), r.URL.Path))
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
		writeProblem(w, r, domainerror.Problem(err, r.URL.Path))
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/api/v1/trips/%s", created.ID))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(trip.ToAPI(created))
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, prob api.Problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(prob)
}
