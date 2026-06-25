package storage

import (
	"errors"
	"sort"
	"strconv"
	"strings"

	models "github.com/bazueva/metrics/internal/model"
)

type Storage interface {
	UpdateMetric(metricType string, name string, value string) error
	GetMetric(name string) (models.Metrics, error)
	GetAllMetrics() []models.Metrics
}

var (
	ErrInvalidMetricType   = errors.New("invalid metric type")
	ErrInvalidGaugeValue   = errors.New("invalid value for gauge")
	ErrInvalidCounterValue = errors.New("invalid value for counter")
	ErrEmptyMetricName     = errors.New("empty metric name")
	ErrInvalidMetricValue  = errors.New("empty value for metric")
	ErrNotFoundMetric      = errors.New("not found")
)

type MemStorage struct {
	metrics map[string]models.Metrics
}

func (ms *MemStorage) GetAllMetrics() []models.Metrics {
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
	metric, found := ms.metrics[metricName]
	if !found {
		return models.Metrics{}, ErrNotFoundMetric
	}

	return metric, nil
}

func (ms *MemStorage) UpdateMetric(metricType string, name string, value string) error {
	name = strings.TrimSpace(name)
	value = strings.TrimSpace(value)

	if name == "" {
		return ErrEmptyMetricName
	}

	if value == "" {
		return ErrInvalidMetricValue
	}

	switch metricType {
	case models.Gauge:
		return ms.addGauge(name, value)
	case models.Counter:
		return ms.addCounter(name, value)
	default:
		return ErrInvalidMetricType
	}
}

func (ms *MemStorage) addGauge(name string, value string) error {
	gauge, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return ErrInvalidGaugeValue
	}

	ms.metrics[name] = models.Metrics{
		ID:    name,
		MType: models.Gauge,
		Value: &gauge,
	}

	return nil
}

func (ms *MemStorage) addCounter(name string, value string) error {
	counter, err := strconv.Atoi(value)
	if err != nil {
		return ErrInvalidCounterValue
	}

	if metric, found := ms.metrics[name]; found {
		*metric.Delta += int64(counter)
	} else {
		ms.metrics[name] = models.Metrics{
			ID:    name,
			MType: models.Counter,
			Delta: new(int64(counter)),
		}
	}

	return nil
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]models.Metrics),
	}
}
