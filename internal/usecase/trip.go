package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/Hil-sys/TripGo-autumn/internal/repository"
)

type TripUsecase struct {
	repo      *repository.TripRepository
	txManager *repository.TxManager
}

func NewTripUsecase(repo *repository.TripRepository, txManager *repository.TxManager) *TripUsecase {
	return &TripUsecase{
		repo:      repo,
		txManager: txManager,
	}
}

type CreateTripParams struct {
	UserID   string
	DriverID string
	StartLat float64
	StartLon float64
	EndLat   float64
	EndLon   float64
	Price    int64
}

func (u *TripUsecase) CreateTrip(ctx context.Context, p CreateTripParams) (*repository.Trip, error) {
	tripID := uuid.New().String()

	trip := &repository.Trip{
		ID:       tripID,
		UserID:   p.UserID,
		DriverID: p.DriverID,
		StartPoint: repository.Point{
			Latitude:  p.StartLat,
			Longitude: p.StartLon,
		},
		EndPoint: repository.Point{
			Latitude:  p.EndLat,
			Longitude: p.EndLon,
		},
		Price:  p.Price,
		Status: repository.TripStatusActive, // Используем типизированную константу
		StartedAt: time.Now().UTC(),
	}

	err := u.txManager.Do(ctx, func(txCtx context.Context) error {
		return u.repo.CreateTrip(txCtx, trip)
	})
	if err != nil {
		return nil, err
	}

	// trip.StartedAt уже заполнен через RETURNING started_at в репозитории
	return trip, nil
}

// GetTrip возвращает поездку по ID
func (u *TripUsecase) GetTrip(ctx context.Context, tripID string) (*repository.Trip, error) {
	return u.repo.GetTrip(ctx, tripID)
}

// FinishTrip атомарно завершает поездку и возвращает обновлённую модель из транзакции
func (u *TripUsecase) FinishTrip(ctx context.Context, tripID string) (*repository.Trip, error) {
	var finishedTrip *repository.Trip

	err := u.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := u.repo.FinishTrip(txCtx, tripID); err != nil {
			return err
		}

		var err error
		finishedTrip, err = u.repo.GetTrip(txCtx, tripID)
		return err
	})
	if err != nil {
		return nil, err
	}

	return finishedTrip, nil
}

func (u *TripUsecase) Ping(ctx context.Context) error {
	return u.repo.Ping(ctx)
}