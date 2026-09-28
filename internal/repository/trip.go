package repository

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgconn"
)

// Определяем кастомные ошибки для ручек по ТЗ
var (
	ErrDriverBusy    = errors.New("driver is busy with another active trip")
	ErrTripNotFound  = errors.New("trip not found")
	ErrTripCompleted = errors.New("trip already completed")
)

type TripRepository struct {
	txManager *TxManager
}

func NewTripRepository(txManager *TxManager) *TripRepository {
	return &TripRepository{txManager: txManager}
}

// CreateTrip атомарно создает поездку (Пункт 5 и 6.3 ТЗ)
func (r *TripRepository) CreateTrip(ctx context.Context, t *Trip) error {
	// Достаем исполнителя (транзакцию или пул) из контекста
	db := r.txManager.GetQueryer(ctx)

	// 1. Вставляем саму поездку в таблицу trips
	sql, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Insert("trips").
		Columns("id", "user_id", "driver_id", "price", "status").
		Values(t.ID, t.UserID, t.DriverID, t.Price, t.Status).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert trip sql: %w", err)
	}

	_, err = db.Exec(ctx, sql, args...)
	if err != nil {
		// Проверяем код ошибки Postgres 23505 (unique_violation)
		// Это означает, что наш частичный уникальный индекс сработал и водитель занят!
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDriverBusy
		}
		return fmt.Errorf("execute insert trip: %w", err)
	}

	// 2. Вставляем стартовый статус в журнал истории (Пункт 4.2 ТЗ)
	sqlHist, argsHist, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Insert("trip_status_history").
		Columns("trip_id", "status").
		Values(t.ID, t.Status).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert history sql: %w", err)
	}

	_, err = db.Exec(ctx, sqlHist, argsHist...)
	if err != nil {
		return fmt.Errorf("execute insert history: %w", err)
	}

	return nil
}
