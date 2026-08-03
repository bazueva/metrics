package metrics

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bazueva/metrics/internal/interfaces"
	models "github.com/bazueva/metrics/internal/model"
	dbPkg "github.com/bazueva/metrics/internal/repository/db"
	"github.com/samber/lo"
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

func (r *Repository) Save(ctx context.Context, data []models.Metrics) error {
	if len(data) == 0 {
		return nil
	}

	chunks := lo.Chunk(data, 100)
	for _, chunk := range chunks {
		args := make([]interface{}, 0, len(chunk)*4)
		sql := `INSERT INTO metrics(metric_id, type, delta, value) VALUES `
		for i, metric := range chunk {
			args = append(args, metric.ID, metric.MType, metric.Delta, metric.Value)
			sql += fmt.Sprintf("($%d, $%d, $%d, $%d)", i*4+1, i*4+2, i*4+3, i*4+4)

			if i != len(chunk)-1 {
				sql += ","
			}
		}

		sql += ` ON CONFLICT (metric_id) DO UPDATE 
		SET type = EXCLUDED.type, 
			delta = EXCLUDED.delta, 
			value = EXCLUDED.value,
			updated_at = CURRENT_TIMESTAMP`

		_, err := r.executeWithRetry(ctx, false, "insert into metrics", sql, args...)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Repository) executeWithRetry(
	ctx context.Context,
	isQuery bool,
	queryName string,
	query string,
	args ...any,
) (any, error) {
	maxRetries := 4

	for attempt := 1; attempt <= maxRetries; attempt++ {
		result, err := func() (any, error) {
			ctxWithTimeout, cancel := context.WithTimeout(ctx, defaultTimeout*time.Second)
			defer cancel()

			if attempt > 1 {
				r.logger.Info("Попытка выполнения запроса",
					zap.String("query name", queryName),
					zap.Int("attempt", attempt),
					zap.String("delay", time.Now().Format("15:04:05.000")),
				)
			}

			if isQuery {
				return r.db.QueryContext(ctxWithTimeout, query, args...)
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

func (r *Repository) Load(ctx context.Context) ([]models.Metrics, error) {
	resultAny, err := r.executeWithRetry(ctx, true, "select all metric", `SELECT metric_id, type, delta, value FROM metrics`)
	if err != nil {
		return nil, err
	}

	rows, ok := resultAny.(*sql.Rows)
	if !ok {
		return nil, fmt.Errorf("Неверный тип результата")
	}

	defer rows.Close()

	result := make([]models.Metrics, 0)
	for rows.Next() {
		var metric models.Metrics
		err = rows.Scan(&metric.ID, &metric.MType, &metric.Delta, &metric.Value)
		if err != nil {
			return nil, err
		}

		result = append(result, metric)
	}

	if err = rows.Err(); err != nil && !errors.Is(err, context.Canceled) {
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
