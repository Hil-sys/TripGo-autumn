package trip

import (
	"context"

	"github.com/Hil-sys/TripGo-autumn/internal/infra/postgres/transaction"
)

type txManager interface {
	GetQueryer(ctx context.Context) transaction.Queryer
	Ping(ctx context.Context) error
}
