//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package create

import (
	"context"

	"github.com/Hil-sys/TripGo-autumn/internal/model"
	usecase "github.com/Hil-sys/TripGo-autumn/internal/usecase/trip/create"
)

type (
	usecaseInterface interface {
		Execute(ctx context.Context, input usecase.Input) (model.Trip, error)
	}

	metricsRecorder interface {
		TripCreated(ctx context.Context)
	}
)
