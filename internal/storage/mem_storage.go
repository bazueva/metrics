package storage

import (
	"errors"
	"sort"
	"strconv"
	"strings"

	models "github.com/bazueva/metrics/internal/model"
)

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

func (ms *MemStorage) addGauge(metric models.Metrics) {
	ms.metrics[metric.ID] = metric
}

func (ms *MemStorage) addCounter(metricData models.Metrics) {
	if metric, found := ms.metrics[metricData.ID]; found {
		*metric.Delta += *metricData.Delta
	} else {
		ms.metrics[metricData.ID] = metricData
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

func NewMemStorage() *MemStorage {
	return &MemStorage{
		metrics: make(map[string]models.Metrics),
	}
}
