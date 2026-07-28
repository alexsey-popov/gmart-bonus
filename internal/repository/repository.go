package repository

import (
	"errors"
	"log/slog"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

var (
	// ErrConflictUnique Ошибка конфликта уникальности при выполнении команды
	ErrConflictUnique = errors.New("ошибка уникальности по полю в БД")
)

// Repository Репозиторий БД
type Repository struct {
	db  *sqlx.DB
	log *slog.Logger
}

// CreateUser Создание нового пользователя в БД
func (rep Repository) CreateUser(login, passwordHash string) (userID string, err error) {
	// Добавляем нового пользователя в БД
	query := `INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id`

	err = rep.db.QueryRow(query, login, passwordHash).Scan(&userID)
	if err != nil {
		// Если произошла ошибка уникальности по полю login - выводим соответствующую ошибку
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation && pgErr.ConstraintName == "idx_users_login_unique" {
			return "", errors.Join(err, ErrConflictUnique)
		}

		rep.log.Error("ошибка при создании нового пользователя в БД", slog.Any("error", err))

		return "", err
	}

	return userID, nil
}

// GetUserIdAndPassword Получение id и пароля пользователя по логину
func (rep Repository) GetUserIdAndPassword(login string) (userID, password string, err error) {
	err = rep.db.QueryRow("SELECT id, password FROM users WHERE login = $1", login).Scan(&userID, &password)

	return
}

// NewRepository Создание нового репозитория
func NewRepository(db *sqlx.DB, log *slog.Logger) *Repository {
	return &Repository{
		db:  db,
		log: log,
	}
}
