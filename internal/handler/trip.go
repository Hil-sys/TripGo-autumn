package handler

import (
	"context"
	"errors"
	"net/http"

	"encoding/json"

	"github.com/google/uuid"

	api "github.com/Hil-sys/TripGo-autumn/internal/api"
	"github.com/Hil-sys/TripGo-autumn/internal/repository"
	openapi_types "github.com/oapi-codegen/runtime/types"
)

// RFC 9457
type Problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance"`
	Code     string `json:"code"`
}

type TripHandler struct {
	repo      *repository.TripRepository
	txManager *repository.TxManager
}

func NewTripHandler(repo *repository.TripRepository, txManager *repository.TxManager) *TripHandler {
	return &TripHandler{
		repo:      repo,
		txManager: txManager,
	}
}

func (h *TripHandler) respondWithError(w http.ResponseWriter, r *http.Request, statusCode int, code, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(statusCode)

	problem := api.Problem{
		Type:     "https://tripgo.example/problems/" + code,
		Title:    title,
		Status:   int32(statusCode),
		Detail:   &detail,
		Instance: &r.URL.Path,
		Code:     code,
	}

	_ = json.NewEncoder(w).Encode(problem)
}

func isInvalidUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err != nil
}

// 1. POST /api/v1/trips - Создание поездки
func (h *TripHandler) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	var req api.CreateTripJSONRequestBody

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, r, http.StatusBadRequest, "invalid_request", "Bad Request", "Invalid JSON body")
		return
	}

	// 2. Превращаем openapi_types.UUID в обычные строки
	userIDStr := req.UserId.String()
	driverIDStr := req.DriverId.String()

	// 3. Валидируем уже строковые значения
	if isInvalidUUID(userIDStr) || isInvalidUUID(driverIDStr) || req.Price < 0 {
		h.respondWithError(w, r, http.StatusBadRequest, "invalid_request", "Bad Request", "Validation failed")
		return
	}

	tripID := uuid.New().String()

	// 4. Перекладываем данные в твою доменную модель
	trip := &repository.Trip{
		ID:       tripID,
		UserID:   userIDStr,
		DriverID: driverIDStr,
		// У структуры Coordinates внутри тоже должны быть поля Latitude и Longitude
		StartPoint: repository.Point{Latitude: req.StartPoint.Latitude, Longitude: req.StartPoint.Longitude},
		EndPoint:   repository.Point{Latitude: req.EndPoint.Latitude, Longitude: req.EndPoint.Longitude},
		Price:      req.Price,
		Status:     "active",
	}

	err := h.txManager.Do(r.Context(), func(ctx context.Context) error {
		return h.repo.CreateTrip(ctx, trip)
	})

	if err != nil {
		if errors.Is(err, repository.ErrDriverBusy) {
			h.respondWithError(w, r, http.StatusConflict, "driver_busy", "Driver Busy", "Driver already has an active trip")
			return
		}
		h.respondWithError(w, r, http.StatusInternalServerError, "internal_error", "Internal Error", "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/api/v1/trips/"+tripID)
	w.WriteHeader(http.StatusCreated)

	apiTripID, _ := uuid.Parse(trip.ID)
	apiUserID, _ := uuid.Parse(trip.UserID)
	apiDriverID, _ := uuid.Parse(trip.DriverID)

	response := api.Trip{
		Id:       openapi_types.UUID(apiTripID),
		UserId:   openapi_types.UUID(apiUserID),
		DriverId: openapi_types.UUID(apiDriverID),
		StartPoint: api.Coordinates{
			Latitude:  trip.StartPoint.Latitude,
			Longitude: trip.StartPoint.Longitude,
		},
		EndPoint: api.Coordinates{
			Latitude:  trip.EndPoint.Latitude,
			Longitude: trip.EndPoint.Longitude,
		},
		Price:      trip.Price,
		Status:     api.TripStatus(trip.Status),
		StartedAt:  trip.StartedAt,
		FinishedAt: trip.FinishedAt,
	}

	_ = json.NewEncoder(w).Encode(response)
}

// 2. GET /api/v1/trips/{tripId} - Получение поездки
func (h *TripHandler) GetTrip(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	tripStr := tripId.String()

	trip, err := h.repo.GetTrip(r.Context(), tripStr)
	if err != nil {
		if errors.Is(err, repository.ErrTripNotFound) {
			h.respondWithError(w, r, http.StatusNotFound, "trip_not_found", "Not Found", "Trip not found")
			return
		}
		h.respondWithError(w, r, http.StatusInternalServerError, "internal_error", "Internal Error", "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(trip)
}

// 3. POST /api/v1/trips/{tripId}/finish - Завершение поездки
func (h *TripHandler) FinishTrip(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	tripStr := tripId.String()

	err := h.txManager.Do(r.Context(), func(ctx context.Context) error {
		return h.repo.FinishTrip(ctx, tripStr)
	})

	if err != nil {
		if errors.Is(err, repository.ErrTripNotFound) {
			h.respondWithError(w, r, http.StatusNotFound, "trip_not_found", "Not Found", "Trip not found")
			return
		}
		if errors.Is(err, repository.ErrTripCompleted) {
			h.respondWithError(w, r, http.StatusConflict, "trip_completed", "Conflict", "Trip already completed")
			return
		}
		h.respondWithError(w, r, http.StatusInternalServerError, "internal_error", "Internal Error", "Internal server error")
		return
	}

	trip, err := h.repo.GetTrip(r.Context(), tripStr)
	if err != nil {
		h.respondWithError(w, r, http.StatusInternalServerError, "internal_error", "Internal Error", "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(trip)
}

// 4. GET /health - Системная ручка
func (h *TripHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"OK"}`))
}

// 5. GET /ready - Системная ручка
func (h *TripHandler) Ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	h.respondWithError(w, r, http.StatusNotImplemented, "not_implemented", "Not Implemented", "Use main ready handler")
}

// Заглушки
func (h *TripHandler) ListTripPositions(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *TripHandler) CreateTripPosition(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	w.WriteHeader(http.StatusNotImplemented)
}
