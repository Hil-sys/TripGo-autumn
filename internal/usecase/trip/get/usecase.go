package get

import (
	"context"

	"github.com/google/uuid"

	domain "github.com/Hil-sys/TripGo-autumn/internal/domain/trip"
	"github.com/Hil-sys/TripGo-autumn/internal/model"
)

type Usecase struct {
	repo tripRepository
}

func New(repo tripRepository) *Usecase {
	return &Usecase{repo: repo}
}

func (u *Usecase) Execute(ctx context.Context, id string) (model.Trip, error) {
	if _, err := uuid.Parse(id); err != nil {
		return model.Trip{}, domain.ValidationError{Field: "id", Reason: "must be a valid UUID"}
	}

	return u.repo.GetByID(ctx, id)
}
