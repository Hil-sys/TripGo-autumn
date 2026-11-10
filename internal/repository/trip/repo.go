package trip

import (
	"context"
	"time"

	"github.com/Hil-sys/TripGo-autumn/internal/infra/telemetry"
)

type Repository struct {
	txManager    txManager
	queryTimeout time.Duration
	metrics      *telemetry.Metrics
}

func NewRepository(tm txManager, queryTimeout time.Duration, metrics *telemetry.Metrics) *Repository {
	return &Repository{
		txManager:    tm,
		queryTimeout: queryTimeout,
		metrics:      metrics,
	}
}

func (r *Repository) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	return r.txManager.Ping(ctx)
}
