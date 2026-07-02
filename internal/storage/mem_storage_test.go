package storage

import (
	"testing"

	models "github.com/bazueva/metrics/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestMemStorage_UpdateMetric(t *testing.T) {
	type args struct {
		metricType  string
		metricName  string
		metricValue string
		setup       func(storage *MemStorage)
	}

	type want struct {
		metrics map[string]models.Metrics
		err     error
	}

	type test struct {
		name string
		args args
		want want
	}

	tests := []test{
		{
			name: "empty metric name",
			args: args{},
			want: want{
				metrics: make(map[string]models.Metrics),
				err:     ErrEmptyMetricName,
			},
		},
		{
			name: "empty metric value",
			args: args{
				metricName: "test",
			},
			want: want{
				err:     ErrInvalidMetricValue,
				metrics: make(map[string]models.Metrics),
			},
		},
		{
			name: "undefined metric type",
			args: args{
				metricType:  "test1",
				metricName:  "test",
				metricValue: "value",
			},
			want: want{
				err:     ErrInvalidMetricType,
				metrics: make(map[string]models.Metrics),
			},
		},
		{
			name: "invalid gauge value",
			args: args{
				metricType:  models.Gauge,
				metricName:  "test",
				metricValue: "value",
			},
			want: want{
				err:     ErrInvalidGaugeValue,
				metrics: make(map[string]models.Metrics),
			},
		},
		{
			name: "invalid counter value",
			args: args{
				metricType:  models.Counter,
				metricName:  "test",
				metricValue: "value",
			},
			want: want{
				err:     ErrInvalidCounterValue,
				metrics: make(map[string]models.Metrics),
			},
		},
		{
			name: "success add counter value",
			args: args{
				metricType:  models.Counter,
				metricName:  "test",
				metricValue: "1",
			},
			want: want{
				metrics: map[string]models.Metrics{
					"test": {
						ID:    "test",
						MType: models.Counter,
						Delta: new(int64(1)),
					},
				},
				err: nil,
			},
		},
		{
			name: "success add gauge value",
			args: args{
				metricType:  models.Gauge,
				metricName:  "test",
				metricValue: "1",
			},
			want: want{
				metrics: map[string]models.Metrics{
					"test": {
						ID:    "test",
						MType: models.Gauge,
						Value: new(float64(1)),
					},
				},
				err: nil,
			},
		},
		{
			name: "gauge accumulation",
			args: args{
				metricType:  models.Gauge,
				metricName:  "test",
				metricValue: "2",
				setup: func(storage *MemStorage) {
					storage.metrics["test"] = models.Metrics{
						ID:    "test",
						MType: models.Gauge,
						Delta: new(int64(1)),
					}
				},
			},
			want: want{
				metrics: map[string]models.Metrics{
					"test": {
						ID:    "test",
						MType: models.Gauge,
						Value: new(float64(2)),
					},
				},
				err: nil,
			},
		},
		{
			name: "counter accumulation",
			args: args{
				metricType:  models.Counter,
				metricName:  "count",
				metricValue: "2",
				setup: func(storage *MemStorage) {
					storage.metrics["count"] = models.Metrics{
						ID:    "count",
						MType: models.Counter,
						Delta: new(int64(1000)),
					}
				},
			},
			want: want{
				metrics: map[string]models.Metrics{
					"count": {
						ID:    "count",
						MType: models.Counter,
						Delta: new(int64(1002)),
					},
				},
				err: nil,
			},
		},
		{
			name: "with spaces",
			args: args{
				metricType:  models.Counter,
				metricName:  " count ",
				metricValue: " 2",
			},
			want: want{
				metrics: map[string]models.Metrics{
					"count": {
						ID:    "count",
						MType: models.Counter,
						Delta: new(int64(2)),
					},
				},
				err: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := NewMemStorage()
			if tt.args.setup != nil {
				tt.args.setup(storage)
			}

			err := storage.UpdateMetric(tt.args.metricType, tt.args.metricName, tt.args.metricValue)

			assert.Equal(t, tt.want.metrics, storage.metrics)

			if tt.want.err != nil {
				assert.ErrorIs(t, err, tt.want.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
