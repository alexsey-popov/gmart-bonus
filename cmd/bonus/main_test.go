package main

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateUp_Error(t *testing.T) {
	t.Run("negative - некорректный запуск миграций", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		_ = db.Close() // close db to cause error in postgres.WithInstance

		sqlxDB := sqlx.NewDb(db, "sqlmock")

		err = migrateUp(sqlxDB)
		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
