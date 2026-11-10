package http

import (
	"net/http"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/Hil-sys/TripGo-autumn/internal/generated"
	systemhandler "github.com/Hil-sys/TripGo-autumn/internal/http/system"
	createhandler "github.com/Hil-sys/TripGo-autumn/internal/http/trip/create"
	finishhandler "github.com/Hil-sys/TripGo-autumn/internal/http/trip/finish"
	gethandler "github.com/Hil-sys/TripGo-autumn/internal/http/trip/get"
)

var _ api.ServerInterface = (*Server)(nil)

type Server struct {
	createHandler *createhandler.Handler
	getHandler    *gethandler.Handler
	finishHandler *finishhandler.Handler
	systemHandler *systemhandler.Handler
}

func NewServer(
	createH *createhandler.Handler,
	getH *gethandler.Handler,
	finishH *finishhandler.Handler,
	systemH *systemhandler.Handler,
) *Server {
	return &Server{
		createHandler: createH,
		getHandler:    getH,
		finishHandler: finishH,
		systemHandler: systemH,
	}
}

func (s *Server) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	s.createHandler.Handle(w, r)
}

func (s *Server) GetTrip(w http.ResponseWriter, r *http.Request, tripId openapi_types.UUID) {
	s.getHandler.Handle(w, r, tripId.String())
}

func (s *Server) FinishTrip(w http.ResponseWriter, r *http.Request, tripId openapi_types.UUID) {
	s.finishHandler.Handle(w, r, tripId.String())
}

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	s.systemHandler.Health(w, r)
}

func (s *Server) Ready(w http.ResponseWriter, r *http.Request) {
	s.systemHandler.Ready(w, r)
}

func (s *Server) CreateTripPosition(w http.ResponseWriter, r *http.Request, _ openapi_types.UUID) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (s *Server) ListTripPositions(w http.ResponseWriter, r *http.Request, _ openapi_types.UUID) {
	w.WriteHeader(http.StatusNotImplemented)
}
