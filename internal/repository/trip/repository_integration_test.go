//go:build integration

package trip

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hil-sys/TripGo-autumn/internal/config"
	domain "github.com/Hil-sys/TripGo-autumn/internal/domain/trip"
	"github.com/Hil-sys/TripGo-autumn/internal/infra/postgres"
	"github.com/Hil-sys/TripGo-autumn/internal/infra/postgres/transaction"
	"github.com/Hil-sys/TripGo-autumn/internal/model"
)

func setupIntegrationTest(t *testing.T) (*pgxpool.Pool, *Repository, *transaction.TxManager) {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// #nosec G101
		dbURL = "postgres://tripgo:tripgo@localhost:5432/tripgo?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg := config.DatabaseConfig{
		URL:             dbURL,
		MaxConns:        5,
		MinConns:        1,
		ConnectTimeout:  3 * time.Second,
		QueryTimeout:    3 * time.Second,
		MaxConnLifetime: time.Minute,
	}

	pool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		t.Skipf("skipping integration test: database not available at %s: %v", dbURL, err)
	}

	tm := transaction.NewTxManager(pool)
	repo := NewRepository(tm, 3*time.Second)

	return pool, repo, tm
}

func TestRepository_Integration_SuccessLifecycle(t *testing.T) {
	pool, repo, tm := setupIntegrationTest(t)

	tripID := uuid.New().String()
	userID := uuid.New().String()
	driverID := uuid.New().String()

	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, "DELETE FROM trip_status_history WHERE trip_id = $1", tripID)
		_, _ = pool.Exec(ctx, "DELETE FROM trips WHERE id = $1", tripID)
		pool.Close()
	})

	ctx := context.Background()

	tr := &model.Trip{
		ID:       tripID,
		UserID:   userID,
		DriverID: driverID,
		StartPoint: model.Point{
			Latitude:  55.751244,
			Longitude: 37.618423,
		},
		EndPoint: model.Point{
			Latitude:  55.761244,
			Longitude: 37.628423,
		},
		Price:  750,
		Status: string(domain.StatusActive),
	}

	// 1. Save trip inside transaction
	err := tm.Do(ctx, func(ctx context.Context) error {
		if err := repo.SaveTrip(ctx, tr); err != nil {
			return err
		}
		fromStatus := string(domain.StatusActive)
		return repo.SaveStatusHistory(ctx, tripID, &fromStatus, string(domain.StatusActive))
	})
	require.NoError(t, err)
	assert.False(t, tr.StartedAt.IsZero())

	// 2. Fetch by ID
	fetched, err := repo.GetByID(ctx, tripID)
	require.NoError(t, err)
	assert.Equal(t, tripID, fetched.ID)
	assert.Equal(t, userID, fetched.UserID)
	assert.Equal(t, driverID, fetched.DriverID)
	assert.Equal(t, string(domain.StatusActive), fetched.Status)

	// 3. Update finished status and save history
	finishedAt := time.Now().UTC()
	err = tm.Do(ctx, func(ctx context.Context) error {
		if err := repo.UpdateFinishedStatus(ctx, tripID, finishedAt); err != nil {
			return err
		}
		fromStatus := string(domain.StatusActive)
		return repo.SaveStatusHistory(ctx, tripID, &fromStatus, string(domain.StatusCompleted))
	})
	require.NoError(t, err)

	// 4. Verify completed status and fetch
	fetchedCompleted, err := repo.GetByID(ctx, tripID)
	require.NoError(t, err)
	assert.Equal(t, string(domain.StatusCompleted), fetchedCompleted.Status)
	require.NotNil(t, fetchedCompleted.FinishedAt)
}

func TestRepository_Integration_RollbackOnHistoryFailure(t *testing.T) {
	pool, repo, tm := setupIntegrationTest(t)

	tripID := uuid.New().String()
	userID := uuid.New().String()
	driverID := uuid.New().String()

	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, "DELETE FROM trip_status_history WHERE trip_id = $1", tripID)
		_, _ = pool.Exec(ctx, "DELETE FROM trips WHERE id = $1", tripID)
		pool.Close()
	})

	ctx := context.Background()

	tr := &model.Trip{
		ID:       tripID,
		UserID:   userID,
		DriverID: driverID,
		StartPoint: model.Point{
			Latitude:  55.751244,
			Longitude: 37.618423,
		},
		EndPoint: model.Point{
			Latitude:  55.761244,
			Longitude: 37.628423,
		},
		Price:  1000,
		Status: string(domain.StatusActive),
	}

	// Attempt transaction where SaveTrip succeeds, but history insertion fails (invalid status check violation)
	err := tm.Do(ctx, func(ctx context.Context) error {
		if err := repo.SaveTrip(ctx, tr); err != nil {
			return err
		}
		fromStatus := string(domain.StatusActive)
		// Passing an invalid status "invalid_status" to trigger a database check constraint violation error
		return repo.SaveStatusHistory(ctx, tripID, &fromStatus, "invalid_status")
	})
	require.Error(t, err)

	// Verify that the trip row was rolled back and does NOT exist in the database
	_, err = repo.GetByID(ctx, tripID)
	assert.ErrorIs(t, err, domain.ErrTripNotFound)
}
