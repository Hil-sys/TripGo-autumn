package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBQueryer объединяет методы выполнения запросов для пула и транзакции
type DBQueryer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type txKey struct{}

type TxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) *TxManager {
	return &TxManager{pool: pool}
}

// Do открывает транзакцию и кладет её в контекст
func (tm *TxManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	// Если транзакция уже открыта выше по стеку — переиспользуем её
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	// 1. Явно задаем уровень изоляции Read Committed через BeginTx
	tx, err := tm.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Хелпер для отката с отдельным независимым контекстом
	rollbackTx := func() error {
		rbCtx, rbCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer rbCancel()
		return tx.Rollback(rbCtx)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = rollbackTx()
			panic(p)
		}
	}()

	ctxWithTx := context.WithValue(ctx, txKey{}, tx)
	if err := fn(ctxWithTx); err != nil {
		// 2. Откатываем транзакцию через отдельный контекст, который не зависит от отмены ctx
		if rbErr := rollbackTx(); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			return fmt.Errorf("fn error: %w, rollback error: %v", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetQueryer достает из контекста транзакцию или возвращает пул
func (tm *TxManager) GetQueryer(ctx context.Context) DBQueryer {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return tm.pool
}

func (tm *TxManager) Ping(ctx context.Context) error {
	return tm.pool.Ping(ctx)
}