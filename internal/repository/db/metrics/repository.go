package metrics

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/bazueva/metrics/internal/interfaces"
	models "github.com/bazueva/metrics/internal/model"
	dbPkg "github.com/bazueva/metrics/internal/repository/db"
	"go.uber.org/zap"
)

const (
	defaultTimeout = 1 * time.Second
)

type Query interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type Repository struct {
	db              Query
	errorClassifier *dbPkg.PostgresErrorClassifier
	logger          interfaces.Logger
}

const chunkSize = 100

func (r *Repository) Save(ctx context.Context, data []models.Metrics) error {
	if len(data) == 0 {
		return nil
	}

	for start := 0; start < len(data); start += chunkSize {
		end := min(start+chunkSize, len(data))
		chunk := data[start:end]

		args := make([]interface{}, 0, len(chunk)*4)

		var queryBuilder strings.Builder
		queryBuilder.WriteString(
			`INSERT INTO metrics(metric_id, type, delta, value) VALUES `,
		)

		for i, metric := range chunk {
			args = append(args, metric.ID, metric.MType, metric.Delta, metric.Value)

			fmt.Fprintf(
				&queryBuilder,
				"($%d, $%d, $%d, $%d)",
				i*4+1,
				i*4+2,
				i*4+3,
				i*4+4,
			)

			if i != len(chunk)-1 {
				queryBuilder.WriteString(",")
			}
		}

		queryBuilder.WriteString(` ON CONFLICT (metric_id) DO UPDATE 
		SET type = EXCLUDED.type, 
			delta = EXCLUDED.delta, 
			value = EXCLUDED.value,
			updated_at = CURRENT_TIMESTAMP`)

		_, err := r.executeWithRetry(
			ctx,
			"insert into metrics",
			queryBuilder.String(),
			args...,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Repository) executeWithRetry(
	ctx context.Context,
	queryName string,
	query string,
	args ...any,
) (sql.Result, error) {
	maxRetries := 4

	for attempt := 1; attempt <= maxRetries; attempt++ {
		result, err := func() (sql.Result, error) {
			ctxWithTimeout, cancel := context.WithTimeout(ctx, defaultTimeout)
			defer cancel()

			if attempt > 1 {
				r.logger.Info("Попытка выполнения запроса",
					zap.String("query name", queryName),
					zap.Int("attempt", attempt),
					zap.String("delay", time.Now().Format("15:04:05.000")),
				)
			}

			return r.db.ExecContext(ctxWithTimeout, query, args...)
		}()

		if err == nil {
			return result, err
		}

		r.logger.Error("Ошибка выполнения запроса",
			zap.Error(err),
		)

		if r.errorClassifier.ClassifyRetry(err) != dbPkg.Retriable {
			return result, err
		}

		if attempt == maxRetries {
			return result, err
		}

		delay := time.Duration(2*attempt-1) * time.Second

		time.Sleep(delay)
	}

	return nil, nil
}

func (r *Repository) queryWithRetry(
	ctx context.Context,
	queryName string,
	scanFn func(*sql.Rows) error,
	query string,
	args ...any,
) error {
	maxRetries := 4

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := func() error {
			ctxWithTimeout, cancel := context.WithTimeout(ctx, defaultTimeout)
			defer cancel()

			if attempt > 1 {
				r.logger.Info("Попытка выполнения запроса",
					zap.String("query name", queryName),
					zap.Int("attempt", attempt),
					zap.String("delay", time.Now().Format("15:04:05.000")),
				)
			}

			rows, err := r.db.QueryContext(ctxWithTimeout, query, args...)
			if err != nil {
				return err
			}
			defer rows.Close()

			if err = scanFn(rows); err != nil {
				return err
			}

			return rows.Err()
		}()

		if err == nil {
			return nil
		}

		r.logger.Error("Ошибка выполнения запроса",
			zap.Error(err),
		)

		if r.errorClassifier.ClassifyRetry(err) != dbPkg.Retriable {
			return err
		}

		if attempt == maxRetries {
			return err
		}

		delay := time.Duration(2*attempt-1) * time.Second
		time.Sleep(delay)
	}

	return fmt.Errorf("unexpected error")
}

func (r *Repository) Load(ctx context.Context) ([]models.Metrics, error) {
	result := make([]models.Metrics, 0)

	err := r.queryWithRetry(
		ctx,
		"select all metric",
		func(rows *sql.Rows) error {
			for rows.Next() {
				var metric models.Metrics
				if err := rows.Scan(&metric.ID, &metric.MType, &metric.Delta, &metric.Value); err != nil {
					return err
				}
				result = append(result, metric)
			}
			return rows.Err()
		},
		`SELECT metric_id, type, delta, value FROM metrics`)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func NewRepository(db Query, logger interfaces.Logger) *Repository {
	return &Repository{
		db:              db,
		logger:          logger,
		errorClassifier: dbPkg.NewPostgresErrorClassifier(),
	}
}
