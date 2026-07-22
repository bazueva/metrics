package storage

import (
	"fmt"
	"testing"
	"time"

	models "github.com/bazueva/metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type FileRepositoryMock struct {
	err       error
	data      []models.Metrics
	callCount int
}

func (f *FileRepositoryMock) Save(data []models.Metrics) error {
	f.data = data
	f.callCount++
	return f.err
}

func (f *FileRepositoryMock) LoadFromFile() ([]models.Metrics, error) {
	return f.data, f.err
}

type LoggerMock struct {
	callCount int
}

func (l *LoggerMock) Error(msg string, fields ...zap.Field) {
	l.callCount++
}

func TestMemStorage_UpdateMetric(t *testing.T) {
	type args struct {
		metric models.Metrics
		setup  func(storage *MemStorage)
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
				metric: models.Metrics{
					ID:    "test",
					MType: models.Gauge,
				},
			},
			want: want{
				err:     ErrInvalidGaugeValue,
				metrics: make(map[string]models.Metrics),
			},
		},
		{
			name: "undefined metric type",
			args: args{
				metric: models.Metrics{
					ID:    "test",
					MType: "test1",
					Delta: nil,
					Value: nil,
					Hash:  "",
				},
			},
			want: want{
				err:     ErrInvalidMetricType,
				metrics: make(map[string]models.Metrics),
			},
		},
		{
			name: "invalid gauge value",
			args: args{
				metric: models.Metrics{
					ID:    "test",
					MType: models.Gauge,
					Delta: nil,
					Value: nil,
					Hash:  "",
				},
			},
			want: want{
				err:     ErrInvalidGaugeValue,
				metrics: make(map[string]models.Metrics),
			},
		},
		{
			name: "invalid counter value",
			args: args{
				metric: models.Metrics{
					ID:    "test",
					MType: models.Counter,
					Delta: nil,
					Value: nil,
					Hash:  "",
				},
			},
			want: want{
				err:     ErrInvalidCounterValue,
				metrics: make(map[string]models.Metrics),
			},
		},
		{
			name: "success add counter value",
			args: args{
				metric: models.Metrics{
					ID:    "test",
					MType: models.Counter,
					Delta: new(int64(1)),
					Value: nil,
					Hash:  "",
				},
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
				metric: models.Metrics{
					ID:    "test",
					MType: models.Gauge,
					Value: new(float64(1)),
					Hash:  "",
				},
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
				metric: models.Metrics{
					ID:    "test",
					MType: models.Gauge,
					Delta: nil,
					Value: new(float64(2)),
				},
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
				metric: models.Metrics{
					ID:    "count",
					MType: models.Counter,
					Delta: new(int64(2)),
				},
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
				metric: models.Metrics{
					ID:    " count ",
					MType: models.Counter,
					Delta: new(int64(2)),
					Value: nil,
				},
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
			storage := NewMemStorage(nil, false, nil, 10)
			if tt.args.setup != nil {
				tt.args.setup(storage)
			}

			err := storage.UpdateMetric(tt.args.metric)

			assert.Equal(t, tt.want.metrics, storage.metrics)

			if tt.want.err != nil {
				assert.ErrorIs(t, err, tt.want.err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMemStorage_CreateMetric(t *testing.T) {
	type args struct {
		metricType  string
		metricName  string
		metricValue string
	}

	type test struct {
		name       string
		args       args
		wantResult models.Metrics
		wantErr    error
	}

	tests := []test{
		{
			name:       "empty data",
			args:       args{},
			wantResult: models.Metrics{},
			wantErr:    ErrInvalidMetricType,
		},
		{
			name: "invalid gauge value",
			args: args{
				metricType:  models.Gauge,
				metricName:  "test",
				metricValue: "",
			},
			wantResult: models.Metrics{},
			wantErr:    ErrInvalidGaugeValue,
		},
		{
			name: "invalid counter value",
			args: args{
				metricType:  models.Counter,
				metricName:  "test",
				metricValue: "",
			},
			wantResult: models.Metrics{},
			wantErr:    ErrInvalidCounterValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MemStorage{}
			metric, err := ms.CreateMetric(tt.args.metricType, tt.args.metricName, tt.args.metricValue)
			if err != nil || tt.wantErr != nil {
				assert.ErrorIs(t, tt.wantErr, err)
			}

			assert.Equal(t, tt.wantResult, metric)
		})
	}
}

func TestMemStorage_RunSaver(t *testing.T) {
	t.Run("storeInterval = 0", func(t *testing.T) {
		fileRepo := &FileRepositoryMock{}
		logger := &LoggerMock{}

		storage := NewMemStorage(fileRepo, false, logger, 0)
		go storage.RunSaver()

		time.Sleep(4 * time.Second)
		assert.Equal(t, 0, fileRepo.callCount)
	})

	t.Run("storeInterval > 0", func(t *testing.T) {
		fileRepo := &FileRepositoryMock{}
		logger := &LoggerMock{}

		storage := NewMemStorage(fileRepo, false, logger, 1)
		go storage.RunSaver()

		time.Sleep(4 * time.Second)
		assert.Greater(t, fileRepo.callCount, 0)
	})
}

func TestMemStorage_Save(t *testing.T) {
	type test struct {
		name     string
		metrics  map[string]models.Metrics
		fileRepo *FileRepositoryMock
		err      string
	}

	tests := []test{
		{
			name: "error repo",
			metrics: map[string]models.Metrics{
				"test": {
					ID:    "test",
					MType: models.Gauge,
					Value: new(float64(1)),
				},
				"test2": {
					ID:    "test2",
					MType: models.Counter,
					Delta: new(int64(2)),
				},
			},
			fileRepo: &FileRepositoryMock{err: fmt.Errorf("Ошибка")},
			err:      "Ошибка",
		},
		{
			name: "success",
			metrics: map[string]models.Metrics{
				"test": {
					ID:    "test",
					MType: models.Gauge,
					Value: new(float64(1)),
				},
				"test2": {
					ID:    "test2",
					MType: models.Counter,
					Delta: new(int64(2)),
				},
			},
			fileRepo: &FileRepositoryMock{},
			err:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &MemStorage{
				metrics:        tt.metrics,
				fileRepository: tt.fileRepo,
			}

			err := storage.Save()
			if err != nil || tt.err != "" {
				assert.Equal(t, tt.err, err.Error())
			}

			metricSlice := make([]models.Metrics, 0, len(tt.metrics))
			for _, metric := range tt.metrics {
				metricSlice = append(metricSlice, metric)
			}

			assert.ElementsMatch(t, metricSlice, tt.fileRepo.data)
		})
	}
}
