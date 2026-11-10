//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package finish

import (
	"context"
	"time"

	"github.com/Hil-sys/TripGo-autumn/internal/model"
)

type (
	tripRepository interface {
		GetByIDForUpdate(ctx context.Context, id string) (model.Trip, error)
		UpdateFinishedStatus(ctx context.Context, id string, finishedAt time.Time) error
		SaveStatusHistory(ctx context.Context, tripID string, fromStatus *string, toStatus string) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}
)
