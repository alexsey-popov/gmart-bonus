package repository

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenericRepository(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("positive - FindAll", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		genRepo := NewGenericRepository[struct {
			ID string `db:"id"`
		}](sqlxDB, log)

		mock.ExpectQuery("SELECT id FROM test").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1").AddRow("2"))

		items, err := genRepo.FindAll(context.Background(), "SELECT id FROM test")
		require.NoError(t, err)
		assert.Len(t, items, 2)
		assert.Equal(t, "1", items[0].ID)
		assert.Equal(t, "2", items[1].ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("negative - FindAll", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		genRepo := NewGenericRepository[struct {
			ID string `db:"id"`
		}](sqlxDB, log)

		mock.ExpectQuery("SELECT id FROM test").
			WillReturnError(errors.New("db error"))

		items, err := genRepo.FindAll(context.Background(), "SELECT id FROM test")
		require.Error(t, err)
		assert.Nil(t, items)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("positive - FindOne", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")
		genRepo := NewGenericRepository[struct {
			ID string `db:"id"`
		}](sqlxDB, log)

		mock.ExpectQuery("SELECT id FROM test").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("1"))

		item, err := genRepo.FindOne(context.Background(), "SELECT id FROM test")
		require.NoError(t, err)
		assert.Equal(t, "1", item.ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
