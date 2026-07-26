package metrics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	models "github.com/bazueva/metrics/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestRepository_Save(t *testing.T) {
	t.Run("empty data", func(t *testing.T) {
		repo := NewRepository(nil)

		assert.Nil(t, repo.Save(nil, nil))
	})

	t.Run("error repo", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		ctx := context.Background()

		repo := NewRepository(db)

		mock.ExpectExec(`INSERT INTO metrics(metric_id, type, delta, value) VALUES ($1, $2, $3, $4),($5, $6, $7, $8) ON CONFLICT (metric_id) DO UPDATE 
    SET type = EXCLUDED.type, 
        delta = EXCLUDED.delta, 
        value = EXCLUDED.value, 
        updated_at = CURRENT_TIMESTAMP`).
			WithArgs(
				"test1", models.Gauge, nil, new(float64(1)),
				"test2", models.Counter, new(int64(5)), nil,
			).
			WillReturnError(errors.New("ошибка"))

		data := []models.Metrics{
			{
				ID:    "test1",
				MType: models.Gauge,
				Value: new(float64(1)),
			},
			{
				ID:    "test2",
				MType: models.Counter,
				Delta: new(int64(5)),
			},
		}
		err = repo.Save(ctx, data)

		assert.Equal(t, "ошибка", err.Error())
	})

	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		ctx := context.Background()

		repo := NewRepository(db)

		mock.ExpectExec(`INSERT INTO metrics(metric_id, type, delta, value) VALUES ($1, $2, $3, $4) ON CONFLICT (metric_id) DO UPDATE 
		SET type = EXCLUDED.type, 
			delta = EXCLUDED.delta, 
			value = EXCLUDED.value,
			updated_at = CURRENT_TIMESTAMP`).
			WithArgs(
				"test1", models.Gauge, nil, new(float64(1)),
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		data := []models.Metrics{
			{
				ID:    "test1",
				MType: models.Gauge,
				Value: new(float64(1)),
			},
		}
		err = repo.Save(ctx, data)
		assert.Nil(t, err)
	})

	t.Run("timeout", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		ctx := context.Background()

		repo := NewRepository(db)

		mock.ExpectExec(`INSERT INTO metrics(metric_id, type, delta, value) VALUES ($1, $2, $3, $4) ON CONFLICT (metric_id) DO UPDATE 
		SET type = EXCLUDED.type, 
			delta = EXCLUDED.delta, 
			value = EXCLUDED.value,
			updated_at = CURRENT_TIMESTAMP`).
			WithArgs(
				"test1", models.Gauge, nil, new(float64(1)),
			).
			WillDelayFor(4 * time.Second).
			WillReturnResult(sqlmock.NewResult(1, 1))

		data := []models.Metrics{
			{
				ID:    "test1",
				MType: models.Gauge,
				Value: new(float64(1)),
			},
		}
		err = repo.Save(ctx, data)
		assert.Equal(t, "canceling query due to user request", err.Error())
	})
}

func TestRepository_Load(t *testing.T) {
	t.Run("error query", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		ctx := context.Background()

		repo := NewRepository(db)

		mock.ExpectQuery(`SELECT metric_id, type, delta, value FROM metrics`).
			WillReturnError(errors.New("ошибка"))

		data, err := repo.Load(ctx)

		assert.Nil(t, data)
		assert.Equal(t, "ошибка", err.Error())
	})

	t.Run("error scan", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		ctx := context.Background()

		repo := NewRepository(db)

		mock.ExpectQuery(`SELECT metric_id, type, delta, value FROM metrics`).
			WillReturnRows(
				sqlmock.NewRows([]string{"metric_id", "type", "delta", "value"}).
					AddRow("test", models.Gauge, 1, "#23"),
			)

		data, err := repo.Load(ctx)

		assert.Nil(t, data)
		assert.Equal(t, "sql: Scan error on column index 3, name \"value\": converting driver.Value type string (\"#23\") to a float64: invalid syntax", err.Error())
	})

	t.Run("error rows", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		ctx := context.Background()

		repo := NewRepository(db)

		mock.ExpectQuery(`SELECT metric_id, type, delta, value FROM metrics`).
			WillReturnRows(
				sqlmock.NewRows([]string{"metric_id", "type", "delta", "value"}).
					AddRow("test", models.Gauge, 1, nil).
					AddRow("test", models.Gauge, 1, nil).
					RowError(1, errors.New("ошибка соединения")),
			)

		data, err := repo.Load(ctx)

		assert.Nil(t, data)
		assert.Equal(t, "ошибка соединения", err.Error())
	})

	t.Run("timeout", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		ctx := context.Background()

		repo := NewRepository(db)

		mock.ExpectQuery(`SELECT metric_id, type, delta, value FROM metrics`).
			WillReturnRows(
				sqlmock.NewRows([]string{"metric_id", "type", "delta", "value"}).
					AddRow("test", models.Gauge, 1, nil),
			).
			WillDelayFor(2 * time.Second)

		data, err := repo.Load(ctx)

		assert.Nil(t, data)
		assert.Equal(t, "canceling query due to user request", err.Error())
	})

	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		ctx := context.Background()

		repo := NewRepository(db)

		mock.ExpectQuery(`SELECT metric_id, type, delta, value FROM metrics`).
			WillReturnRows(
				sqlmock.NewRows([]string{"metric_id", "type", "delta", "value"}).
					AddRow("test", models.Gauge, 1, nil).
					AddRow("test2", models.Counter, nil, 6.5),
			)

		data, err := repo.Load(ctx)

		assert.Nil(t, err)
		assert.Equal(t, []models.Metrics{
			{
				ID:    "test",
				MType: models.Gauge,
				Delta: new(int64(1)),
			},
			{
				ID:    "test2",
				MType: models.Counter,
				Delta: nil,
				Value: new(float64(6.5)),
			},
		}, data)
	})
}
