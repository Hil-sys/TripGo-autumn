package finish

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

func (u *Usecase) Execute(ctx context.Context, id string) (model.Trip, error) {
	if _, err := uuid.Parse(id); err != nil {
		return model.Trip{}, domain.ValidationError{Field: "id", Reason: "must be a valid UUID"}
	}

	var completedTrip model.Trip

	err := u.txManager.Do(ctx, func(txCtx context.Context) error {
		currentTrip, err := u.repo.GetByIDForUpdate(txCtx, id)
		if err != nil {
			return err
		}

		dTrip := domain.Trip{
			ID:         currentTrip.ID,
			UserID:     currentTrip.UserID,
			DriverID:   currentTrip.DriverID,
			StartPoint: domain.Point{Latitude: currentTrip.StartPoint.Latitude, Longitude: currentTrip.StartPoint.Longitude},
			EndPoint:   domain.Point{Latitude: currentTrip.EndPoint.Latitude, Longitude: currentTrip.EndPoint.Longitude},
			Price:      currentTrip.Price,
			Status:     domain.Status(currentTrip.Status),
			StartedAt:  currentTrip.StartedAt,
			FinishedAt: currentTrip.FinishedAt,
		}

		finishTime := time.Now().UTC()
		if err := dTrip.Finish(finishTime); err != nil {
			return err

		if err := u.repo.UpdateFinishedStatus(txCtx, id, finishTime); err != nil {
			return fmt.Errorf("update finished status: %w", err)
		}

		fromStatus := currentTrip.Status
		if err := u.repo.SaveStatusHistory(txCtx, id, &fromStatus, string(domain.StatusCompleted)); err != nil {
			return fmt.Errorf("save status history: %w", err)
		}

		currentTrip.Status = string(domain.StatusCompleted)
		currentTrip.FinishedAt = &finishTime
		completedTrip = currentTrip

		return nil
	})
	if err != nil {
		return model.Trip{}, err
	}

	return completedTrip, nil
}
