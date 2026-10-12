package metrics

import (
	"context"
	"database/sql"
	"strconv"
	"testing"

	models "github.com/bazueva/metrics/internal/model"
	"go.uber.org/zap"
)

type benchmarkQuery struct{}

func (q *benchmarkQuery) ExecContext(
	ctx context.Context,
	query string,
	args ...any,
) (sql.Result, error) {
	return nil, nil
}

func (q *benchmarkQuery) QueryContext(
	ctx context.Context,
	query string,
	args ...any,
) (*sql.Rows, error) {
	return nil, nil
}

func BenchmarkRepository_Save(b *testing.B) {
	tests := []struct {
		name  string
		count int
	}{
		{
			name:  "10_metrics",
			count: 10,
		},
		{
			name:  "100_metrics",
			count: 100,
		},
		{
			name:  "1000_metrics",
			count: 1000,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			repository := NewRepository(
				&benchmarkQuery{},
				zap.NewNop(),
			)

			data := make([]models.Metrics, 0, tt.count)

			for i := 0; i < tt.count; i++ {
				value := float64(i)

				data = append(data, models.Metrics{
					ID:    strconv.Itoa(i),
					MType: models.Gauge,
					Value: &value,
				})
			}

			ctx := context.Background()

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if err := repository.Save(ctx, data); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
