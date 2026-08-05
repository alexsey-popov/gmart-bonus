package repository

import (
	"log/slog"

	"github.com/jmoiron/sqlx"
)

// Repository Репозиторий БД
type Repository struct {
	db  *sqlx.DB
	log *slog.Logger
}

// NewRepository Создание нового репозитория
func NewRepository(db *sqlx.DB, log *slog.Logger) *Repository {
	return &Repository{
		db:  db,
		log: log,
	}
}
