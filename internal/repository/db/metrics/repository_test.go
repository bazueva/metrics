package metrics

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bazueva/metrics/internal/interfaces/mocks"
	models "github.com/bazueva/metrics/internal/model"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	mock2 "github.com/stretchr/testify/mock"
)

func TestRepository_Save(t *testing.T) {
	t.Run("empty data", func(t *testing.T) {
		repo := NewRepository(nil, nil)

		assert.Nil(t, repo.Save(nil, nil))
	})

	t.Run("error repo", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		logger := mocks.NewMockLogger(t)
		logger.EXPECT().Error("Ошибка выполнения запроса", mock2.Anything)

		ctx := context.Background()

		repo := NewRepository(db, logger)

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

	t.Run("retry", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		ctx := context.Background()

		logger := mocks.NewMockLogger(t)
		logger.EXPECT().
			Error("Ошибка выполнения запроса", mock2.Anything).
			Times(4)

		logger.EXPECT().
			Info("Попытка выполнения запроса", mock2.Anything).
			Times(3)

		repo := NewRepository(db, logger)

		connectErr := &pgconn.ConnectError{
			Config: &pgconn.Config{
				Host: "127.0.0.1",
				Port: 5432,
			},
		}

		val := reflect.ValueOf(connectErr).Elem()
		field := val.FieldByName("err")

		field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()

		internalErr := errors.New("connection refused")
		field.Set(reflect.ValueOf(internalErr))

		queryStr := `INSERT INTO metrics(metric_id, type, delta, value) VALUES ($1, $2, $3, $4),($5, $6, $7, $8) ON CONFLICT (metric_id) DO UPDATE 
    SET type = EXCLUDED.type, 
        delta = EXCLUDED.delta, 
        value = EXCLUDED.value, 
        updated_at = CURRENT_TIMESTAMP`
		for i := 0; i < 4; i++ {
			mock.ExpectExec(queryStr).
				WithArgs(
					"test1", models.Gauge, nil, new(float64(1)),
					"test2", models.Counter, new(int64(5)), nil,
				).
				WillReturnError(connectErr)
		}

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

		assert.Equal(t, "failed to connect to `user= database=`: connection refused", err.Error())
	})

	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		ctx := context.Background()

		repo := NewRepository(db, nil)

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

		logger := mocks.NewMockLogger(t)
		logger.EXPECT().Error("Ошибка выполнения запроса", mock2.Anything)

		repo := NewRepository(db, logger)

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

		logger := mocks.NewMockLogger(t)
		logger.EXPECT().Error("Ошибка выполнения запроса", mock2.Anything)

		repo := NewRepository(db, logger)

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

		repo := NewRepository(db, nil)

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

		repo := NewRepository(db, nil)

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

		logger := mocks.NewMockLogger(t)
		logger.EXPECT().Error("Ошибка выполнения запроса", mock2.Anything)

		repo := NewRepository(db, logger)

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

	t.Run("retry", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		ctx := context.Background()

		logger := mocks.NewMockLogger(t)
		logger.EXPECT().
			Error("Ошибка выполнения запроса", mock2.Anything).
			Times(4)

		logger.EXPECT().
			Info("Попытка выполнения запроса", mock2.Anything).
			Times(3)

		repo := NewRepository(db, logger)

		connectErr := &pgconn.ConnectError{
			Config: &pgconn.Config{
				Host: "127.0.0.1",
				Port: 5432,
			},
		}

		val := reflect.ValueOf(connectErr).Elem()
		field := val.FieldByName("err")

		field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()

		internalErr := errors.New("connection refused")
		field.Set(reflect.ValueOf(internalErr))

		queryStr := `SELECT metric_id, type, delta, value FROM metrics`
		for i := 0; i < 4; i++ {
			mock.ExpectQuery(queryStr).WillReturnError(connectErr)
		}

		data, err := repo.Load(ctx)

		assert.Nil(t, data)
		assert.Equal(t, "failed to connect to `user= database=`: connection refused", err.Error())
	})

	t.Run("success", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()

		ctx := context.Background()

		repo := NewRepository(db, nil)

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
