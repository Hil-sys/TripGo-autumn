package trip

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgconn"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	domain "github.com/Hil-sys/TripGo-autumn/internal/domain/trip"
	"github.com/Hil-sys/TripGo-autumn/internal/model"
)

// SaveTrip выполняет вставку новой поездки в таблицу trips.
func (r *Repository) SaveTrip(ctx context.Context, t *model.Trip) error {
	start := time.Now()
	var opErr error
	defer func() {
		if r.metrics != nil && r.metrics.DBDuration != nil {
			errType := ""
			if opErr != nil {
				errType = fmt.Sprintf("%T", opErr)
			}
			r.metrics.DBDuration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(
				attribute.String("db.operation.name", "INSERT"),
				attribute.String("db.collection.name", "trips"),
				attribute.String("error.type", errType),
			))
		}
	}()

	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	q := r.txManager.GetQueryer(ctx)

	query, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Insert("trips").
		Columns("id", "user_id", "driver_id", "start_latitude", "start_longitude",
			"end_latitude", "end_longitude", "price", "status", "started_at").
		Values(t.ID, t.UserID, t.DriverID, t.StartPoint.Latitude, t.StartPoint.Longitude,
			t.EndPoint.Latitude, t.EndPoint.Longitude, t.Price, t.Status, t.StartedAt).
		Suffix("RETURNING started_at").
		ToSql()
	if err != nil {
		opErr = err
		return fmt.Errorf("build insert trip sql: %w", err)
	}

	err = q.QueryRow(ctx, query, args...).Scan(&t.StartedAt)
	if err != nil {
		opErr = err
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrDriverBusy
		}
		return fmt.Errorf("execute insert trip: %w", err)
	}

	return nil
}

// SaveStatusHistory записывает переход статуса в журнал trip_status_history.
func (r *Repository) SaveStatusHistory(ctx context.Context, tripID string, fromStatus *string, toStatus string) error {
	start := time.Now()
	var opErr error
	defer func() {
		if r.metrics != nil && r.metrics.DBDuration != nil {
			errType := ""
			if opErr != nil {
				errType = fmt.Sprintf("%T", opErr)
			}
			r.metrics.DBDuration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(
				attribute.String("db.operation.name", "INSERT"),
				attribute.String("db.collection.name", "trip_status_history"),
				attribute.String("error.type", errType),
			))
		}
	}()

	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	q := r.txManager.GetQueryer(ctx)

	query, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status").
		Values(tripID, fromStatus, toStatus).
		ToSql()
	if err != nil {
		opErr = err
		return fmt.Errorf("build insert history sql: %w", err)
	}

	if _, err := q.Exec(ctx, query, args...); err != nil {
		opErr = err
		return fmt.Errorf("execute insert history: %w", err)
	}

	return nil
}

// UpdateFinishedStatus фиксирует статус completed и время завершения.
func (r *Repository) UpdateFinishedStatus(ctx context.Context, id string, finishedAt time.Time) error {
	start := time.Now()
	var opErr error
	defer func() {
		if r.metrics != nil && r.metrics.DBDuration != nil {
			errType := ""
			if opErr != nil {
				errType = fmt.Sprintf("%T", opErr)
			}
			r.metrics.DBDuration.Record(ctx, time.Since(start).Seconds(), metric.WithAttributes(
				attribute.String("db.operation.name", "UPDATE"),
				attribute.String("db.collection.name", "trips"),
				attribute.String("error.type", errType),
			))
		}
	}()

	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	q := r.txManager.GetQueryer(ctx)

	query, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Update("trips").
		Set("status", string(domain.StatusCompleted)).
		Set("finished_at", finishedAt).
		Set("updated_at", finishedAt).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		opErr = err
		return fmt.Errorf("build update trip sql: %w", err)
	}

	if _, err := q.Exec(ctx, query, args...); err != nil {
		opErr = err
		return fmt.Errorf("execute update trip: %w", err)
	}

	return nil
}
