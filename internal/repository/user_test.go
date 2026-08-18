package repository

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateUser Проверка метода создания пользователя в репозитории
func TestCreateUser(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("positive - успешное создание пользователя", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		login := "testuser"
		passwordHash := "hashedpass"
		expectedID := "user-uuid-123"

		mock.ExpectQuery(`INSERT INTO users \(login, password\) VALUES \(\$1, \$2\) RETURNING id`).
			WithArgs(login, passwordHash).
			WillReturnRows(
				sqlmock.NewRows([]string{"id"}).AddRow(expectedID),
			)

		userID, err := rep.CreateUser(context.Background(), login, passwordHash)
		require.NoError(t, err)
		assert.Equal(t, expectedID, userID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("negative - конфликт уникальности (UniqueViolation)", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		login := "existinguser"
		passwordHash := "hashedpass"

		pgErr := &pgconn.PgError{
			Code:           pgerrcode.UniqueViolation,
			ConstraintName: "idx_users_login_unique",
		}

		mock.ExpectQuery(`INSERT INTO users \(login, password\) VALUES \(\$1, \$2\) RETURNING id`).
			WithArgs(login, passwordHash).
			WillReturnError(pgErr)

		userID, err := rep.CreateUser(context.Background(), login, passwordHash)
		require.Error(t, err)
		assert.Empty(t, userID)
		assert.True(t, errors.Is(err, ErrConflictUnique))
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("negative - другая ошибка БД при создании пользователя", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		login := "testuser"
		passwordHash := "hashedpass"
		dbErr := errors.New("db connection error")

		mock.ExpectQuery(`INSERT INTO users \(login, password\) VALUES \(\$1, \$2\) RETURNING id`).
			WithArgs(login, passwordHash).
			WillReturnError(dbErr)

		userID, err := rep.CreateUser(context.Background(), login, passwordHash)
		require.Error(t, err)
		assert.Empty(t, userID)
		assert.False(t, errors.Is(err, ErrConflictUnique))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

// TestGetUserIdAndPassword Проверка получения ID и пароля пользователя по логину
func TestGetUserIdAndPassword(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("positive - успешное получение id и пароля", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		login := "testuser"
		expectedID := "user-uuid-123"
		expectedPassword := "hashedpass"

		mock.ExpectQuery(`SELECT id, password FROM users WHERE login = \$1`).
			WithArgs(login).
			WillReturnRows(sqlmock.NewRows([]string{"id", "password"}).AddRow(expectedID, expectedPassword))

		userID, password, err := rep.GetUserIdAndPassword(context.Background(), login)
		require.NoError(t, err)
		assert.Equal(t, expectedID, userID)
		assert.Equal(t, expectedPassword, password)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("negative - пользователь не найден", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		rep := NewRepository(sqlxDB, log)

		login := "unknownuser"

		mock.ExpectQuery(`SELECT id, password FROM users WHERE login = \$1`).
			WithArgs(login).
			WillReturnError(sql.ErrNoRows)

		userID, password, err := rep.GetUserIdAndPassword(context.Background(), login)
		require.Error(t, err)
		assert.Empty(t, userID)
		assert.Empty(t, password)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
