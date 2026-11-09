package create

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	domain "github.com/Hil-sys/TripGo-autumn/internal/domain/trip"
	"github.com/Hil-sys/TripGo-autumn/internal/model"
)

type Usecase struct {
	repo      tripRepository
	txManager txManager
}

func New(repo tripRepository, tm txManager) *Usecase {
	return &Usecase{
		repo:      repo,
		txManager: tm,
	}
}

type Input struct {
	UserID     string
	DriverID   string
	StartPoint model.Point
	EndPoint   model.Point
	Price      int64
}

func (u *Usecase) Execute(ctx context.Context, input Input) (model.Trip, error) {
	userUUID, err := uuid.Parse(input.UserID)
	if err != nil || userUUID == uuid.Nil {
		return model.Trip{}, domain.ValidationError{Field: "user_id", Reason: "must be a valid UUID"}
	}

	driverUUID, err := uuid.Parse(input.DriverID)
	if err != nil || driverUUID == uuid.Nil {
		return model.Trip{}, domain.ValidationError{Field: "driver_id", Reason: "must be a valid UUID"}
	}

	err = domain.ValidateCreate(
		userUUID,
		driverUUID,
		domain.Point{Latitude: input.StartPoint.Latitude, Longitude: input.StartPoint.Longitude},
		domain.Point{Latitude: input.EndPoint.Latitude, Longitude: input.EndPoint.Longitude},
		input.Price,
	)
	if err != nil {
		return model.Trip{}, err
	}

	trip := model.Trip{
		ID:         uuid.New().String(),
		UserID:     input.UserID,
		DriverID:   input.DriverID,
		StartPoint: input.StartPoint,
		EndPoint:   input.EndPoint,
		Price:      input.Price,
		Status:     string(domain.StatusActive),
		StartedAt:  time.Now().UTC(),
	}

	err = u.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := u.repo.SaveTrip(txCtx, &trip); err != nil {
			return fmt.Errorf("save trip: %w", err)
		}

		if err := u.repo.SaveStatusHistory(txCtx, trip.ID, nil, trip.Status); err != nil {
			return fmt.Errorf("save status history: %w", err)
		}

		return nil
	})
	if err != nil {
		return model.Trip{}, err
	}

	return trip, nil
}
