package trip

import (
	"context"
	"time"
)

type Repository struct {
	txManager    txManager
	queryTimeout time.Duration
}

func NewRepository(tm txManager, queryTimeout time.Duration) *Repository {
	return &Repository{
		txManager:    tm,
		queryTimeout: queryTimeout,
	}
}

func (r *Repository) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	return r.txManager.Ping(ctx)
}
