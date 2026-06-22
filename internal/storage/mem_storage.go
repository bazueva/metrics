package storage

import (
	"errors"
	"strconv"
	"strings"

	models "github.com/bazueva/metrics/internal/model"
)

type Storage interface {
	UpdateMetric(metricType string, name string, value string) error
}

var (
	ErrInvalidMetricType   = errors.New("invalid metric type")
	ErrInvalidGaugeValue   = errors.New("invalid value for gauge")
	ErrInvalidCounterValue = errors.New("invalid value for counter")
	ErrEmptyMetricName     = errors.New("empty metric name")
	ErrInvalidMetricValue  = errors.New("empty value for metric")
)

type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
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

	ms.gauges[name] = gauge

	return nil
}

func (ms *MemStorage) addCounter(name string, value string) error {
	counter, err := strconv.Atoi(value)
	if err != nil {
		return ErrInvalidCounterValue
	}

	ms.counters[name] += int64(counter)

	return nil
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}
