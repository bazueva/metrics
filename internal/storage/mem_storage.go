package storage

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bazueva/metrics/internal/interfaces"
	models "github.com/bazueva/metrics/internal/model"
	"go.uber.org/zap"
)

var (
	// ErrInvalidMetricType возвращается при неизвестном типе метрики.
	ErrInvalidMetricType = errors.New("invalid metric type")

	// ErrInvalidGaugeValue возвращается при некорректном значении gauge-метрики.
	ErrInvalidGaugeValue = errors.New("invalid value for gauge")

	// ErrInvalidCounterValue возвращается при некорректном значении counter-метрики.
	ErrInvalidCounterValue = errors.New("invalid value for counter")

	// ErrEmptyMetricName возвращается, если имя метрики не указано.
	ErrEmptyMetricName = errors.New("empty metric name")

	// ErrInvalidMetricValue возвращается при отсутствии значения метрики.
	ErrInvalidMetricValue = errors.New("empty value for metric")

	// ErrNotFoundMetric возвращается, если метрика не найдена.
	ErrNotFoundMetric = errors.New("not found")
)

// Repository определяет методы для сохранения и загрузки метрик.
type Repository interface {
	Save(ctx context.Context, data []models.Metrics) error
	Load(ctx context.Context) ([]models.Metrics, error)
}

// MemStorage реализует потокобезопасное хранение метрик в памяти.
type MemStorage struct {
	metrics       map[string]models.Metrics
	repository    Repository
	logger        interfaces.Logger
	storeInterval int
	mu            sync.RWMutex
}

// UpdatesMetrics обновляет набор метрик и сохраняет изменения.
func (ms *MemStorage) UpdatesMetrics(metrics []models.Metrics) error {
	for _, metric := range metrics {
		if err := ms.validateMetric(metric); err != nil {
			return err
		}
	}

	for _, metric := range metrics {
		if err := ms.UpdateMetric(metric, false); err != nil {
			return err
		}
	}

	return ms.Save()
}

// CreateMetric создаёт метрику указанного типа из строкового значения.
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

// UpdateMetric обновляет значение метрики в хранилище.
// Если needSave установлен в true и периодическое сохранение отключено,
// изменения сохраняются в репозитории.
func (ms *MemStorage) UpdateMetric(metric models.Metrics, needSave bool) error {
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

	if needSave && ms.storeInterval == 0 {
		err := ms.Save()
		if err != nil {
			ms.logger.Error(err.Error())
		}
	}

	return nil
}

// GetAllMetrics возвращает все метрики.
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

// GetMetric возвращает метрику по её имени.
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
}

func (ms *MemStorage) addCounter(metricData models.Metrics) {
	ms.mu.Lock()

	if metric, found := ms.metrics[metricData.ID]; found {
		*metric.Delta += *metricData.Delta
	} else {
		ms.metrics[metricData.ID] = metricData
	}

	ms.mu.Unlock()
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

// Load загружает метрики из репозитория в хранилище.
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

// Save сохраняет текущие метрики из хранилища в репозиторий.
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

	sort.Slice(data, func(i, j int) bool {
		return data[i].ID < data[j].ID
	})

	return ms.repository.Save(context.Background(), data)
}

// RunSaver запускает периодическое сохранение метрик.
// При завершении контекста выполняется последнее сохранение.
func (ms *MemStorage) RunSaver(ctx context.Context) {
	interval := ms.storeInterval

	if interval == 0 {
		return
	}

	go func() {
		tick := time.Tick(time.Duration(ms.storeInterval) * time.Second)

		for {
			select {
			case <-tick:
				err := ms.Save()
				if err != nil {
					ms.logger.Error("Ошибка сохранения метрик", zap.Error(err))
				}
			case <-ctx.Done():
				err := ms.Save()
				if err != nil {
					ms.logger.Error("Ошибка сохранения метрик", zap.Error(err))
				}

				return
			}
		}
	}()
}

// NewMemStorage создаёт новое хранилище метрик.
// Если loadMetrics установлен в true, метрики загружаются из репозитория
// при создании хранилища.
func NewMemStorage(repository Repository, loadMetrics bool, logger interfaces.Logger, storeInterval int) *MemStorage {
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
