package storage

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	models "github.com/bazueva/metrics/internal/model"
	"go.uber.org/zap"
)

var (
	ErrInvalidMetricType   = errors.New("invalid metric type")
	ErrInvalidGaugeValue   = errors.New("invalid value for gauge")
	ErrInvalidCounterValue = errors.New("invalid value for counter")
	ErrEmptyMetricName     = errors.New("empty metric name")
	ErrInvalidMetricValue  = errors.New("empty value for metric")
	ErrNotFoundMetric      = errors.New("not found")
)

type Logger interface {
	Error(msg string, fields ...zap.Field)
}

type Repository interface {
	Save(ctx context.Context, data []models.Metrics) error
	Load(ctx context.Context) ([]models.Metrics, error)
}

type MemStorage struct {
	metrics       map[string]models.Metrics
	repository    Repository
	logger        Logger
	storeInterval int
	mu            sync.RWMutex
}

func (ms *MemStorage) CreateMetric(metricType string, name string, value string) (models.Metrics, error) {
	switch metricType {
	case models.Gauge:
		gauge, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return models.Metrics{}, ErrInvalidGaugeValue
		}

		return models.Metrics{
			ID:    name,
			MType: metricType,
			Value: &gauge,
		}, nil
	case models.Counter:
		counter, err := strconv.Atoi(value)
		if err != nil {
			return models.Metrics{}, ErrInvalidCounterValue
		}

		return models.Metrics{
			ID:    name,
			MType: metricType,
			Delta: new(int64(counter)),
		}, nil
	default:
		return models.Metrics{}, ErrInvalidMetricType
	}
}

func (ms *MemStorage) UpdateMetric(metric models.Metrics) error {
	metric.ID = strings.TrimSpace(metric.ID)
	if err := ms.validateMetric(metric); err != nil {
		return err
	}

	switch metric.MType {
	case models.Gauge:
		ms.addGauge(metric)
	case models.Counter:
		ms.addCounter(metric)
	default:
		return ErrInvalidMetricType
	}

	return nil
}

func (ms *MemStorage) GetAllMetrics() []models.Metrics {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	result := make([]models.Metrics, 0, len(ms.metrics))

	for _, metric := range ms.metrics {
		result = append(result, metric)
	}

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})

	return result
}

func (ms *MemStorage) GetMetric(metricName string) (models.Metrics, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	metric, found := ms.metrics[metricName]
	if !found {
		return models.Metrics{}, ErrNotFoundMetric
	}

	return metric, nil
}

func (ms *MemStorage) addGauge(metric models.Metrics) {
	ms.mu.Lock()
	ms.metrics[metric.ID] = metric
	ms.mu.Unlock()

	if ms.storeInterval == 0 {
		err := ms.Save()
		if err != nil {
			ms.logger.Error(err.Error())
		}
	}
}

func (ms *MemStorage) addCounter(metricData models.Metrics) {
	ms.mu.Lock()

	if metric, found := ms.metrics[metricData.ID]; found {
		*metric.Delta += *metricData.Delta
	} else {
		ms.metrics[metricData.ID] = metricData
	}

	ms.mu.Unlock()

	if ms.storeInterval == 0 {
		err := ms.Save()
		if err != nil {
			ms.logger.Error(err.Error())
		}
	}
}

func (ms *MemStorage) validateMetric(metric models.Metrics) error {
	if metric.ID == "" {
		return ErrEmptyMetricName
	}

	switch metric.MType {
	case models.Gauge:
		if metric.Value == nil {
			return ErrInvalidGaugeValue
		}
	case models.Counter:
		if metric.Delta == nil {
			return ErrInvalidCounterValue
		}
	default:
		return ErrInvalidMetricType
	}

	return nil
}

func (ms *MemStorage) Load() error {
	if ms.repository == nil {
		return nil
	}

	data, err := ms.repository.Load(context.Background())
	if err != nil {
		return err
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.metrics = make(map[string]models.Metrics)
	for _, metric := range data {
		ms.metrics[metric.ID] = metric
	}

	return nil
}

func (ms *MemStorage) Save() error {
	if ms.repository == nil {
		return nil
	}

	ms.mu.RLock()
	defer ms.mu.RUnlock()
	data := make([]models.Metrics, 0, len(ms.metrics))

	for _, metric := range ms.metrics {
		data = append(data, metric)
	}

	return ms.repository.Save(context.Background(), data)
}

func (ms *MemStorage) RunSaver() {
	if ms.storeInterval == 0 {
		return
	}

	go func() {
		for {
			time.Sleep(time.Duration(ms.storeInterval) * time.Second)

			err := ms.Save()
			if err != nil {
				ms.logger.Error("Ошибка сохранения метрик", zap.Error(err))
			}
		}
	}()
}

func NewMemStorage(repository Repository, loadMetrics bool, logger Logger, storeInterval int) *MemStorage {
	storage := &MemStorage{
		metrics:       make(map[string]models.Metrics),
		repository:    repository,
		storeInterval: storeInterval,
		logger:        logger,
	}

	if loadMetrics {
		err := storage.Load()
		if err != nil {
			logger.Error("Ошибка загрузки метрик", zap.Error(err))
		}
	}

	return storage
}
