package metrics

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	models "github.com/bazueva/metrics/internal/model"
	"github.com/samber/lo"
)

const (
	defaultTimeout = 5 * time.Second
	loadTimeout    = 1 * time.Second
)

type Query interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type Repository struct {
	db Query
}

func (r *Repository) Save(ctx context.Context, data []models.Metrics) error {
	if len(data) == 0 {
		return nil
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

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

		_, err := r.db.ExecContext(ctxWithTimeout, sql, args...)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Repository) Load(ctx context.Context) ([]models.Metrics, error) {
	return nil, nil
	ctxWithTimeout, cancel := context.WithTimeout(ctx, loadTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(
		ctxWithTimeout,
		`SELECT metric_id, type, delta, value FROM metrics`,
	)
	defer rows.Close()

	if err != nil {
		return nil, err
	}

	result := make([]models.Metrics, 0)
	for rows.Next() {
		var metric models.Metrics
		err = rows.Scan(&metric.ID, &metric.MType, &metric.Delta, &metric.Value)
		if err != nil {
			return nil, err
		}

		result = append(result, metric)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func NewRepository(db Query) *Repository {
	return &Repository{db: db}
}
