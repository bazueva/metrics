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
		gauges   map[string]float64
		counters map[string]int64
		err      error
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
				gauges:   make(map[string]float64),
				counters: make(map[string]int64),
				err:      ErrEmptyMetricName,
			},
		},
		{
			name: "empty metric value",
			args: args{
				metricName: "test",
			},
			want: want{
				err:      ErrInvalidMetricValue,
				gauges:   make(map[string]float64),
				counters: make(map[string]int64),
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
				err:      ErrInvalidMetricType,
				gauges:   make(map[string]float64),
				counters: make(map[string]int64),
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
				err:      ErrInvalidGaugeValue,
				gauges:   make(map[string]float64),
				counters: make(map[string]int64),
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
				err:      ErrInvalidCounterValue,
				gauges:   make(map[string]float64),
				counters: make(map[string]int64),
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
				gauges:   make(map[string]float64),
				counters: map[string]int64{"test": 1},
				err:      nil,
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
				gauges:   map[string]float64{"test": 1},
				counters: make(map[string]int64),
				err:      nil,
			},
		},
		{
			name: "gauge accumulation",
			args: args{
				metricType:  models.Gauge,
				metricName:  "test",
				metricValue: "2",
				setup: func(storage *MemStorage) {
					storage.gauges["test"] = 1
				},
			},
			want: want{
				gauges:   map[string]float64{"test": 2},
				counters: make(map[string]int64),
				err:      nil,
			},
		},
		{
			name: "counter accumulation",
			args: args{
				metricType:  models.Counter,
				metricName:  "count",
				metricValue: "2",
				setup: func(storage *MemStorage) {
					storage.counters["count"] = 1000
				},
			},
			want: want{
				gauges:   make(map[string]float64),
				counters: map[string]int64{"count": 1002},
				err:      nil,
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
				gauges:   make(map[string]float64),
				counters: map[string]int64{"count": 2},
				err:      nil,
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

			assert.Equal(t, tt.want.counters, storage.counters)
			assert.Equal(t, tt.want.gauges, storage.gauges)

			if tt.want.err != nil {
				assert.ErrorIs(t, err, tt.want.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
