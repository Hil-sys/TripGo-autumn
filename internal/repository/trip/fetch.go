package trip

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"

	domain "github.com/Hil-sys/TripGo-autumn/internal/domain/trip"
	"github.com/Hil-sys/TripGo-autumn/internal/model"
)

func (r *Repository) GetByID(ctx context.Context, id string) (model.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	q := r.txManager.GetQueryer(ctx)

	query, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select("id", "user_id", "driver_id", "start_latitude", "start_longitude",
			"end_latitude", "end_longitude", "price", "status", "started_at", "finished_at").
		From("trips").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return model.Trip{}, fmt.Errorf("build select trip sql: %w", err)
	}

	var row dbTripRow
	err = q.QueryRow(ctx, query, args...).Scan(
		&row.ID, &row.UserID, &row.DriverID,
		&row.StartLatitude, &row.StartLongitude,
		&row.EndLatitude, &row.EndLongitude,
		&row.Price, &row.Status, &row.StartedAt, &row.FinishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Trip{}, domain.ErrTripNotFound
		}
		return model.Trip{}, fmt.Errorf("scan trip row: %w", err)
	}

	return toModelTrip(row), nil
}

func (r *Repository) GetByIDForUpdate(ctx context.Context, id string) (model.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	q := r.txManager.GetQueryer(ctx)

	query, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select("id", "user_id", "driver_id", "start_latitude", "start_longitude",
			"end_latitude", "end_longitude", "price", "status", "started_at", "finished_at").
		From("trips").
		Where(sq.Eq{"id": id}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return model.Trip{}, fmt.Errorf("build select for update sql: %w", err)
	}

	var row dbTripRow
	err = q.QueryRow(ctx, query, args...).Scan(
		&row.ID, &row.UserID, &row.DriverID,
		&row.StartLatitude, &row.StartLongitude,
		&row.EndLatitude, &row.EndLongitude,
		&row.Price, &row.Status, &row.StartedAt, &row.FinishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Trip{}, domain.ErrTripNotFound
		}
		return model.Trip{}, fmt.Errorf("scan for update row: %w", err)
	}

	return toModelTrip(row), nil
}
