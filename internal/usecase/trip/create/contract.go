//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package create

import (
	"context"

	"github.com/Hil-sys/TripGo-autumn/internal/model"
)

type (
	tripRepository interface {
		SaveTrip(ctx context.Context, trip *model.Trip) error
		SaveStatusHistory(ctx context.Context, tripID string, fromStatus *string, toStatus string) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}
)
