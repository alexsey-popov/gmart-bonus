package repository

import (
	"context"
	"log/slog"

	"github.com/jmoiron/sqlx"
)

// GenericRepository Общий generic-репозиторий для CRUD-операций
type GenericRepository[T any] struct {
	db  *sqlx.DB
	log *slog.Logger
}

// NewGenericRepository Создание нового generic-репозитория
func NewGenericRepository[T any](db *sqlx.DB, log *slog.Logger) *GenericRepository[T] {
	return &GenericRepository[T]{
		db:  db,
		log: log,
	}
}

// FindAll Получение списка записей (SELECT)
func (r GenericRepository[T]) FindAll(ctx context.Context, query string, args ...any) (items []T, err error) {
	err = r.db.SelectContext(ctx, &items, query, args...)
	if err != nil {
		r.log.Error("ошибка при получении списка записей", slog.Any("error", err))
	}
	return items, err
}

// FindOne Получение одной записи (GET)
func (r GenericRepository[T]) FindOne(ctx context.Context, query string, args ...any) (item T, err error) {
	err = r.db.GetContext(ctx, &item, query, args...)
	if err != nil {
		r.log.Error("ошибка при получении записи", slog.Any("error", err))
	}
	return item, err
}

// Insert Вставка или запрос с возвратом модели
func (r GenericRepository[T]) Insert(ctx context.Context, query string, args ...any) (item T, err error) {
	err = r.db.GetContext(ctx, &item, query, args...)
	if err != nil {
		r.log.Error("ошибка при выполнении операции вставки/запроса", slog.Any("error", err))
	}
	return item, err
}
