package handler

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	models "github.com/bazueva/metrics/internal/model"
	"go.uber.org/zap"
)

type benchmarkStorage struct {
	metrics []models.Metrics
}

func (s *benchmarkStorage) GetAllMetrics() []models.Metrics {
	return s.metrics
}

func (s *benchmarkStorage) GetMetric(name string) (models.Metrics, error) {
	return models.Metrics{}, nil
}

func (s *benchmarkStorage) UpdateMetric(metric models.Metrics, needSave bool) error {
	return nil
}

func (s *benchmarkStorage) CreateMetric(
	metricType string,
	name string,
	value string,
) (models.Metrics, error) {
	return models.Metrics{}, nil
}

func (s *benchmarkStorage) UpdatesMetrics(metrics []models.Metrics) error {
	return nil
}

func BenchmarkHandler_GetAllMetricsHandler(b *testing.B) {
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
			metrics := make([]models.Metrics, 0, tt.count)

			for i := 0; i < tt.count; i++ {
				value := float64(i)

				metrics = append(metrics, models.Metrics{
					ID:    strconv.Itoa(i),
					MType: models.Gauge,
					Value: &value,
				})
			}

			storage := &benchmarkStorage{
				metrics: metrics,
			}

			h := NewHandler(
				storage,
				zap.NewNop(),
				nil,
				nil,
			)

			request := httptest.NewRequest(
				http.MethodGet,
				"/",
				nil,
			)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				recorder := httptest.NewRecorder()

				h.GetAllMetricsHandler(recorder, request)
			}
		})
	}
}
