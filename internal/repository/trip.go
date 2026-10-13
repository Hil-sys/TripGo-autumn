package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrDriverBusy    = errors.New("driver is busy with another active trip")
	ErrTripNotFound  = errors.New("trip not found")
	ErrTripCompleted = errors.New("trip already completed")
)

type TripRepository struct {
	txManager    *TxManager
	queryTimeout time.Duration
}

func NewTripRepository(txManager *TxManager, queryTimeout time.Duration) *TripRepository {
	return &TripRepository{
		txManager:    txManager,
		queryTimeout: queryTimeout,
	}
}

// CreateTrip атомарно создает поездку и возвращает фактический started_at из БД
func (r *TripRepository) CreateTrip(ctx context.Context, t *Trip) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	queryer := r.txManager.GetQueryer(ctx)

	// 1. Вставляем поездку и через RETURNING started_at вычитываем сгенерированное базой время
	sql, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Insert("trips").
		Columns("id", "user_id", "driver_id", "start_latitude", "start_longitude", "end_latitude", "end_longitude", "price", "status", "started_at").
		Values(t.ID, t.UserID, t.DriverID, t.StartPoint.Latitude, t.StartPoint.Longitude, t.EndPoint.Latitude, t.EndPoint.Longitude, t.Price, t.Status, t.StartedAt).
		Suffix("RETURNING started_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert trip sql: %w", err)
	}

	err = queryer.QueryRow(ctx, sql, args...).Scan(&t.StartedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDriverBusy
		}
		return fmt.Errorf("execute insert trip: %w", err)
	}

	// 2. Логируем статус в trip_status_history по контракту (from_status = NULL, to_status = active)
	sqlHist, argsHist, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status").
		Values(t.ID, nil, t.Status).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert history sql: %w", err)
	}

	_, err = queryer.Exec(ctx, sqlHist, argsHist...)
	if err != nil {
		return fmt.Errorf("execute insert history: %w", err)
	}

	return nil
}

// GetTrip возвращает поездку целиком по её ID
func (r *TripRepository) GetTrip(ctx context.Context, id string) (*Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	queryer := r.txManager.GetQueryer(ctx)

	sql, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select("id", "user_id", "driver_id", "start_latitude", "start_longitude", "end_latitude", "end_longitude", "price", "status", "started_at", "finished_at").
		From("trips").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select trip sql: %w", err)
	}

	var t Trip
	row := queryer.QueryRow(ctx, sql, args...)

	err = row.Scan(
		&t.ID, &t.UserID, &t.DriverID,
		&t.StartPoint.Latitude, &t.StartPoint.Longitude,
		&t.EndPoint.Latitude, &t.EndPoint.Longitude,
		&t.Price, &t.Status, &t.StartedAt, &t.FinishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTripNotFound // 404
		}
		return nil, fmt.Errorf("scan trip row: %w", err)
	}

	return &t, nil
}

// FinishTrip завершает поездку с защитой от гонки через SELECT ... FOR UPDATE
func (r *TripRepository) FinishTrip(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	queryer := r.txManager.GetQueryer(ctx)

	// Блокируем строку поездки для текущей транзакции
	selectSql, selectArgs, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select("status").
		From("trips").
		Where(sq.Eq{"id": id}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return fmt.Errorf("build select for update sql: %w", err)
	}

	var currentStatus TripStatus
	err = queryer.QueryRow(ctx, selectSql, selectArgs...).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrTripNotFound
		}
		return fmt.Errorf("execute select for update: %w", err)
	}

	if currentStatus == TripStatusCompleted {
		return ErrTripCompleted
	}

	// Обновляем статус и выставляем finished_at
	updateSql, updateArgs, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Update("trips").
		Set("status", TripStatusCompleted).
		Set("finished_at", sq.Expr("CURRENT_TIMESTAMP")).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update trip sql: %w", err)
	}

	_, err = queryer.Exec(ctx, updateSql, updateArgs...)
	if err != nil {
		return fmt.Errorf("execute update trip: %w", err)
	}

	// Записываем историю перехода active -> completed
	histSql, histArgs, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status").
		Values(id, TripStatusActive, TripStatusCompleted).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert history finish sql: %w", err)
	}

	_, err = queryer.Exec(ctx, histSql, histArgs...)
	if err != nil {
		return fmt.Errorf("execute insert history finish: %w", err)
	}

	return nil
}

func (r *TripRepository) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	return r.txManager.Ping(ctx)
}
