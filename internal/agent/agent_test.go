package agent

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bazueva/metrics/internal/agent/mocks"
	interfacesMocks "github.com/bazueva/metrics/internal/interfaces/mocks"
	models "github.com/bazueva/metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const testTimeout = 5 * time.Second

func waitForTestSignal(t *testing.T, ch <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(testTimeout):
		t.Fatal(message)
	}
}

func TestExtendedMetricUpdater(t *testing.T) {
	t.Run("ошибка collector", func(t *testing.T) {
		t.Parallel()

		collector := mocks.NewMockCollector(t)
		logger := interfacesMocks.NewMockLogger(t)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		ch := make(chan models.Metrics, 10)
		collected := make(chan struct{})
		done := make(chan struct{})
		var once sync.Once

		collector.EXPECT().ExtendedMetricSnapshot().
			RunAndReturn(func() ([]models.Metrics, error) {
				once.Do(func() {
					cancel()
					close(collected)
				})
				return nil, errors.New("ошибка ExtendedMetricSnapshot")
			})
		logger.EXPECT().Error("Ошибка сборка extended метрик", mock.Anything)

		a := agent{collector: collector, pollInterval: 1, logger: logger}
		go func() {
			defer close(done)
			a.extendedMetricUpdater(ctx, ch)
		}()

		waitForTestSignal(t, collected, "ExtendedMetricSnapshot не был вызван")
		waitForTestSignal(t, done, "extendedMetricUpdater не завершился")
		close(ch)
		assert.Empty(t, drainMetrics(ch))
	})

	t.Run("успешный сбор метрик", func(t *testing.T) {
		t.Parallel()

		collector := mocks.NewMockCollector(t)
		logger := interfacesMocks.NewMockLogger(t)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		metrics := []models.Metrics{
			{ID: "test1", MType: models.Gauge, Value: new(5.6)},
			{ID: "test2", MType: models.Gauge, Value: new(9.6)},
		}
		ch := make(chan models.Metrics, len(metrics))
		done := make(chan struct{})
		collector.EXPECT().ExtendedMetricSnapshot().Return(metrics, nil).Maybe()

		a := agent{collector: collector, pollInterval: 1, logger: logger}
		go func() {
			defer close(done)
			a.extendedMetricUpdater(ctx, ch)
		}()

		actual := make([]models.Metrics, 0, len(metrics))
		for range metrics {
			select {
			case metric := <-ch:
				actual = append(actual, metric)
			case <-time.After(testTimeout):
				cancel()
				waitForTestSignal(t, done, "extendedMetricUpdater не завершился")
				t.Fatal("не удалось получить собранные метрики")
			}
		}
		cancel()
		waitForTestSignal(t, done, "extendedMetricUpdater не завершился")
		assert.Equal(t, metrics, actual)
	})

	t.Run("завершается при отмене контекста", func(t *testing.T) {
		t.Parallel()

		collector := mocks.NewMockCollector(t)
		logger := interfacesMocks.NewMockLogger(t)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		ch := make(chan models.Metrics, 10)
		done := make(chan struct{})
		a := agent{collector: collector, pollInterval: 5, logger: logger}
		go func() {
			defer close(done)
			a.extendedMetricUpdater(ctx, ch)
		}()

		cancel()
		waitForTestSignal(t, done, "extendedMetricUpdater не завершился после отмены контекста")
		close(ch)
		assert.Empty(t, drainMetrics(ch))
	})
}

func TestRuntimeMetricUpdater(t *testing.T) {
	t.Run("успешный сбор метрик", func(t *testing.T) {
		t.Parallel()

		collector := mocks.NewMockCollector(t)
		logger := interfacesMocks.NewMockLogger(t)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		metrics := []models.Metrics{
			{ID: "test1", MType: models.Gauge, Value: new(5.6)},
			{ID: "test2", MType: models.Gauge, Value: new(9.6)},
		}
		ch := make(chan models.Metrics, len(metrics))
		done := make(chan struct{})
		collector.EXPECT().MetricsSnapshot(mock.Anything).Return(metrics).Maybe()

		a := agent{collector: collector, pollInterval: 1, logger: logger}
		go func() {
			defer close(done)
			a.runtimeMetricUpdater(ctx, ch)
		}()

		actual := make([]models.Metrics, 0, len(metrics))
		for range metrics {
			select {
			case metric := <-ch:
				actual = append(actual, metric)
			case <-time.After(testTimeout):
				cancel()
				waitForTestSignal(t, done, "runtimeMetricUpdater не завершился")
				t.Fatal("не удалось получить собранные метрики")
			}
		}
		cancel()
		waitForTestSignal(t, done, "runtimeMetricUpdater не завершился")
		assert.Equal(t, metrics, actual)
	})

	t.Run("завершается при отмене контекста", func(t *testing.T) {
		t.Parallel()

		collector := mocks.NewMockCollector(t)
		logger := interfacesMocks.NewMockLogger(t)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		ch := make(chan models.Metrics, 10)
		done := make(chan struct{})
		a := agent{collector: collector, pollInterval: 5, logger: logger}
		go func() {
			defer close(done)
			a.runtimeMetricUpdater(ctx, ch)
		}()

		cancel()
		waitForTestSignal(t, done, "runtimeMetricUpdater не завершился после отмены контекста")
		close(ch)
		assert.Empty(t, drainMetrics(ch))
	})
}

func TestSenderSnapshot(t *testing.T) {
	t.Run("накопление метрик и отправка по таймеру", func(t *testing.T) {
		t.Parallel()

		repository := mocks.NewMockSenderRepository(t)
		logger := interfacesMocks.NewMockLogger(t)
		metrics := []models.Metrics{
			{ID: "test0", MType: models.Gauge, Value: new(42.0)},
			{ID: "test1", MType: models.Gauge, Value: new(42.0)},
			{ID: "test2", MType: models.Gauge, Value: new(42.0)},
		}
		sent := make(chan struct{})
		repository.EXPECT().SendBatchMetric(metrics).
			RunAndReturn(func([]models.Metrics) error {
				close(sent)
				return nil
			})

		a := agent{repository: repository, reportInterval: 1, logger: logger}
		ch := make(chan models.Metrics, len(metrics))
		done := make(chan struct{})
		for _, metric := range metrics {
			ch <- metric
		}
		go func() {
			defer close(done)
			a.senderSnapshot(ch)
		}()

		waitForTestSignal(t, sent, "метрики не отправлены по таймеру")
		close(ch)
		waitForTestSignal(t, done, "senderSnapshot не завершился после закрытия канала")
	})

	t.Run("отправка остатков при закрытии канала", func(t *testing.T) {
		t.Parallel()

		repository := mocks.NewMockSenderRepository(t)
		logger := interfacesMocks.NewMockLogger(t)
		metrics := []models.Metrics{
			{ID: "test1", MType: models.Gauge, Value: new(42.0)},
			{ID: "test2", MType: models.Gauge, Value: new(43.0)},
		}
		repository.EXPECT().SendBatchMetric(metrics).Return(nil)

		a := agent{repository: repository, reportInterval: 10, logger: logger}
		ch := make(chan models.Metrics, len(metrics))
		done := make(chan struct{})
		for _, metric := range metrics {
			ch <- metric
		}
		close(ch)
		go func() {
			defer close(done)
			a.senderSnapshot(ch)
		}()

		waitForTestSignal(t, done, "senderSnapshot не завершился после закрытия канала")
	})

	t.Run("не отправляет пустой батч", func(t *testing.T) {
		t.Parallel()

		repository := mocks.NewMockSenderRepository(t)
		logger := interfacesMocks.NewMockLogger(t)
		a := agent{repository: repository, reportInterval: 1, logger: logger}
		ch := make(chan models.Metrics)
		done := make(chan struct{})
		go func() {
			defer close(done)
			a.senderSnapshot(ch)
		}()

		time.Sleep(1500 * time.Millisecond)
		close(ch)
		waitForTestSignal(t, done, "senderSnapshot не завершился после закрытия канала")
		repository.AssertNotCalled(t, "SendBatchMetric", mock.Anything)
	})

	t.Run("ошибка отправки логируется", func(t *testing.T) {
		t.Parallel()

		repository := mocks.NewMockSenderRepository(t)
		logger := interfacesMocks.NewMockLogger(t)
		metrics := []models.Metrics{
			{ID: "test1", MType: models.Gauge, Value: new(42.0)},
			{ID: "test2", MType: models.Gauge, Value: new(43.0)},
		}
		sent := make(chan struct{})
		repository.EXPECT().SendBatchMetric(metrics).
			RunAndReturn(func([]models.Metrics) error {
				close(sent)
				return errors.New("ошибка отправки")
			})
		logger.EXPECT().Error("Ошибка отправки метрик", mock.Anything)

		a := agent{repository: repository, reportInterval: 1, logger: logger}
		ch := make(chan models.Metrics, len(metrics))
		done := make(chan struct{})
		for _, metric := range metrics {
			ch <- metric
		}
		go func() {
			defer close(done)
			a.senderSnapshot(ch)
		}()

		waitForTestSignal(t, sent, "отправка метрик не началась")
		close(ch)
		waitForTestSignal(t, done, "senderSnapshot не завершился после закрытия канала")
	})
}

func TestSendMetricToChannel_CancelWhenFull(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	ch := make(chan models.Metrics, 1)
	ch <- models.Metrics{ID: "existing", MType: models.Gauge, Value: new(1.0)}

	a := agent{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		a.sendMetricToChannel(ctx, ch, []models.Metrics{
			{ID: "new", MType: models.Gauge, Value: new(2.0)},
		})
	}()

	cancel()
	waitForTestSignal(t, done, "sendMetricToChannel заблокировался после отмены контекста")
	assert.Len(t, ch, 1)
}

func TestAgent_GracefulShutdown(t *testing.T) {
	t.Parallel()

	collector := mocks.NewMockCollector(t)
	repository := mocks.NewMockSenderRepository(t)
	logger := interfacesMocks.NewMockLogger(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	sendingStarted := make(chan struct{})
	finishSending := make(chan struct{})
	runDone := make(chan struct{})
	var releaseSending sync.Once
	unblockSending := func() { releaseSending.Do(func() { close(finishSending) }) }
	defer unblockSending()

	metrics := []models.Metrics{
		{ID: "test1", MType: models.Gauge, Value: new(5.6)},
		{ID: "test2", MType: models.Gauge, Value: new(9.6)},
	}

	var collected sync.Once
	collector.EXPECT().MetricsSnapshot(mock.Anything).
		RunAndReturn(func(int64) []models.Metrics {
			result := []models.Metrics(nil)
			collected.Do(func() { result = metrics })
			return result
		}).Maybe()
	collector.EXPECT().ExtendedMetricSnapshot().Return(nil, nil).Maybe()

	repository.EXPECT().SendBatchMetric(metrics).
		RunAndReturn(func([]models.Metrics) error {
			close(sendingStarted)
			<-finishSending
			return nil
		})

	a := NewAgent(collector, repository, 1, 1, 1, logger)
	go func() {
		defer close(runDone)
		a.Run(ctx)
	}()

	waitForTestSignal(t, sendingStarted, "отправка метрик не началась")
	cancel()

	select {
	case <-runDone:
		t.Fatal("агент завершился до окончания отправки метрик")
	default:
	}

	unblockSending()
	waitForTestSignal(t, runDone, "агент не завершился после отправки метрик")
}

func drainMetrics(ch <-chan models.Metrics) []models.Metrics {
	var result []models.Metrics
	for metric := range ch {
		result = append(result, metric)
	}
	return result
}
