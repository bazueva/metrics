package agent

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/bazueva/metrics/internal/agent/mocks"
	interfacesMocks "github.com/bazueva/metrics/internal/interfaces/mocks"
	models "github.com/bazueva/metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type SenderRepositoryMock struct {
	err       error
	callCount int
}

func (s *SenderRepositoryMock) SendBatchMetric(metrics []models.Metrics) error {
	s.callCount++
	return s.err
}

func (s *SenderRepositoryMock) SendMetric(metric models.Metrics) error {
	s.callCount++
	return s.err
}

type MetricsSnapshotMock struct {
	metrics   []models.Metrics
	callCount int
}

func (m *MetricsSnapshotMock) MetricsSnapshot(counter int64) []models.Metrics {
	m.callCount++
	return m.metrics
}

func TestExtendedMetricUpdater(t *testing.T) {
	t.Run("ошибка collector", func(t *testing.T) {
		collector := mocks.NewMockCollector(t)
		logger := interfacesMocks.NewMockLogger(t)

		ch := make(chan models.Metrics, 10)

		ctxWithCancel, cancel := context.WithCancel(t.Context())
		defer cancel()

		collector.EXPECT().
			ExtendedMetricSnapshot().
			Return(nil, fmt.Errorf("ошибка ExtendedMetricSnapshot"))

		logger.EXPECT().
			Error("Ошибка сборка extended метрик", mock.Anything)

		agentTest := agent{
			collector:    collector,
			pollInterval: 1,
			logger:       logger,
		}
		go agentTest.extendedMetricUpdater(ctxWithCancel, ch)

		time.Sleep(time.Second * 2)
		cancel()
		time.Sleep(100 * time.Millisecond)

		close(ch)
		var received []models.Metrics
		for m := range ch {
			received = append(received, m)
		}

		assert.Equal(t, 0, len(received))
	})

	t.Run("успешный сбор метрик", func(t *testing.T) {
		collector := mocks.NewMockCollector(t)
		logger := interfacesMocks.NewMockLogger(t)

		ch := make(chan models.Metrics, 10)

		ctxWithCancel, cancel := context.WithCancel(t.Context())
		defer cancel()

		collector.EXPECT().
			ExtendedMetricSnapshot().
			Return([]models.Metrics{
				{
					ID:    "test1",
					MType: models.Gauge,
					Value: new(5.6),
				},
				{
					ID:    "test2",
					MType: models.Gauge,
					Value: new(9.6),
				},
			}, nil).
			Times(2)

		agentTest := agent{
			collector:    collector,
			pollInterval: 1,
			logger:       logger,
		}
		go agentTest.extendedMetricUpdater(ctxWithCancel, ch)

		time.Sleep(time.Millisecond * 2500)
		cancel()
		time.Sleep(100 * time.Millisecond)

		close(ch)
		var received []models.Metrics
		for m := range ch {
			received = append(received, m)
		}

		assert.Equal(t, 4, len(received))
	})

	t.Run("завершается при отмене контекста", func(t *testing.T) {
		collector := mocks.NewMockCollector(t)
		logger := interfacesMocks.NewMockLogger(t)

		ch := make(chan models.Metrics, 10)

		ctxWithCancel, cancel := context.WithCancel(t.Context())
		defer cancel()

		agentTest := agent{
			collector:    collector,
			pollInterval: 5,
			logger:       logger,
		}
		go agentTest.extendedMetricUpdater(ctxWithCancel, ch)

		time.Sleep(time.Millisecond * 100)
		cancel()
		time.Sleep(100 * time.Millisecond)

		close(ch)
		var received []models.Metrics
		for m := range ch {
			received = append(received, m)
		}

		assert.Equal(t, 0, len(received))
	})
}

func TestRuntimeMetricUpdater(t *testing.T) {
	t.Run("успешный сбор метрик", func(t *testing.T) {
		collector := mocks.NewMockCollector(t)
		logger := interfacesMocks.NewMockLogger(t)

		ch := make(chan models.Metrics, 10)

		ctxWithCancel, cancel := context.WithCancel(t.Context())
		defer cancel()

		collector.EXPECT().
			MetricsSnapshot(mock.Anything).
			Return([]models.Metrics{
				{
					ID:    "test1",
					MType: models.Gauge,
					Value: new(5.6),
				},
				{
					ID:    "test2",
					MType: models.Gauge,
					Value: new(9.6),
				},
			}).
			Times(2)

		agentTest := agent{
			collector:    collector,
			pollInterval: 1,
			logger:       logger,
		}
		go agentTest.runtimeMetricUpdater(ctxWithCancel, ch)

		time.Sleep(time.Millisecond * 2500)
		cancel()
		time.Sleep(100 * time.Millisecond)

		close(ch)
		var received []models.Metrics
		for m := range ch {
			received = append(received, m)
		}

		assert.Equal(t, 4, len(received))
	})

	t.Run("завершается при отмене контекста", func(t *testing.T) {
		collector := mocks.NewMockCollector(t)
		logger := interfacesMocks.NewMockLogger(t)

		ch := make(chan models.Metrics, 10)

		ctxWithCancel, cancel := context.WithCancel(t.Context())
		defer cancel()

		agentTest := agent{
			collector:    collector,
			pollInterval: 5,
			logger:       logger,
		}
		go agentTest.runtimeMetricUpdater(ctxWithCancel, ch)

		time.Sleep(time.Millisecond * 100)
		cancel()
		time.Sleep(100 * time.Millisecond)

		close(ch)
		var received []models.Metrics
		for m := range ch {
			received = append(received, m)
		}

		assert.Equal(t, 0, len(received))
	})
}

func TestSenderSnapshot(t *testing.T) {
	t.Run("накопление метрик и отправка по таймеру", func(t *testing.T) {
		repository := mocks.NewMockSenderRepository(t)
		logger := interfacesMocks.NewMockLogger(t)

		repository.EXPECT().
			SendBatchMetric(mock.MatchedBy(func(metrics []models.Metrics) bool {
				return len(metrics) == 3
			})).
			Return(nil).
			Times(1)

		testAgent := &agent{
			repository:     repository,
			reportInterval: 1,
			logger:         logger,
		}

		metricCh := make(chan models.Metrics, 10)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go testAgent.senderSnapshot(ctx, metricCh)

		for i := 0; i < 3; i++ {
			metricCh <- models.Metrics{
				ID:    fmt.Sprintf("test%d", i),
				MType: models.Gauge,
				Value: new(42.0),
			}
		}

		time.Sleep(1500 * time.Millisecond)
		cancel()
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("отправка остатков при закрытии канала", func(t *testing.T) {
		repository := mocks.NewMockSenderRepository(t)
		logger := interfacesMocks.NewMockLogger(t)

		repository.EXPECT().
			SendBatchMetric(mock.MatchedBy(func(metrics []models.Metrics) bool {
				return len(metrics) == 2
			})).
			Return(nil).
			Times(1)

		testAgent := &agent{
			repository:     repository,
			reportInterval: 10,
			logger:         logger,
		}

		metricCh := make(chan models.Metrics, 10)
		ctx := context.Background()

		go testAgent.senderSnapshot(ctx, metricCh)

		metricCh <- models.Metrics{ID: "test1", MType: "gauge", Value: new(42.0)}
		metricCh <- models.Metrics{ID: "test2", MType: "gauge", Value: new(43.0)}

		close(metricCh)
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("не отправляет пустой батч", func(t *testing.T) {
		repository := mocks.NewMockSenderRepository(t)
		logger := interfacesMocks.NewMockLogger(t)

		testAgent := &agent{
			repository:     repository,
			reportInterval: 1,
			logger:         logger,
		}

		metricCh := make(chan models.Metrics, 10)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go testAgent.senderSnapshot(ctx, metricCh)

		time.Sleep(1500 * time.Millisecond)
		cancel()
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("завершение по контексту с отправкой остатков", func(t *testing.T) {
		repository := mocks.NewMockSenderRepository(t)
		logger := interfacesMocks.NewMockLogger(t)

		repository.EXPECT().
			SendBatchMetric(mock.MatchedBy(func(metrics []models.Metrics) bool {
				return len(metrics) == 2
			})).
			Return(nil).
			Times(1)

		testAgent := &agent{
			repository:     repository,
			reportInterval: 10,
			logger:         logger,
		}

		metricCh := make(chan models.Metrics, 10)
		ctx, cancel := context.WithCancel(context.Background())

		go testAgent.senderSnapshot(ctx, metricCh)

		metricCh <- models.Metrics{ID: "test1", MType: "gauge", Value: new(42.0)}
		metricCh <- models.Metrics{ID: "test2", MType: "gauge", Value: new(43.0)}

		time.Sleep(100 * time.Millisecond)
		cancel()
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("ошибка отправки логируется", func(t *testing.T) {
		repository := mocks.NewMockSenderRepository(t)
		logger := interfacesMocks.NewMockLogger(t)

		repository.EXPECT().
			SendBatchMetric(mock.Anything).
			Return(errors.New("ошибка отправки")).
			Times(1)

		logger.EXPECT().
			Error("Ошибка отправки метрик", mock.Anything).
			Times(1)

		testAgent := &agent{
			repository:     repository,
			reportInterval: 1,
			logger:         logger,
		}

		metricCh := make(chan models.Metrics, 10)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go testAgent.senderSnapshot(ctx, metricCh)

		metricCh <- models.Metrics{ID: "test1", MType: "gauge", Value: new(42.0)}
		metricCh <- models.Metrics{ID: "test2", MType: "gauge", Value: new(43.0)}

		time.Sleep(1500 * time.Millisecond)
		cancel()
		time.Sleep(100 * time.Millisecond)
	})
}
