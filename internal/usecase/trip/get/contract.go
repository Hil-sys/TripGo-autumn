//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package get

import (
	"context"

	"github.com/Hil-sys/TripGo-autumn/internal/model"
)

type (
	tripRepository interface {
		GetByID(ctx context.Context, id string) (model.Trip, error)
	}
)
