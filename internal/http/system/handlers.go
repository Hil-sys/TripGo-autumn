package system

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Hil-sys/TripGo-autumn/internal/generated"
)

type pinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	db pinger
}

func NewHandler(db pinger) *Handler {
	return &Handler{db: db}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(api.HealthResponse{Status: "ok"})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := h.db.Ping(r.Context()); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(api.HealthResponse{Status: "unavailable"})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(api.HealthResponse{Status: "ok"})
}
