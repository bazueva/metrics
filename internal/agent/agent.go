package agent

import (
	"context"
	"sync"
	"time"

	"github.com/bazueva/metrics/internal/interfaces"
	models "github.com/bazueva/metrics/internal/model"
	"go.uber.org/zap"
)

const PollInterval = 2
const ReportInterval = 10

type Collector interface {
	MetricsSnapshot(counter int64) []models.Metrics
	ExtendedMetricSnapshot() ([]models.Metrics, error)
}

type SenderRepository interface {
	SendBatchMetric(metrics []models.Metrics) error
}

type agent struct {
	collector      Collector
	repository     SenderRepository
	metrics        []models.Metrics
	reportInterval int
	pollInterval   int
	rateLimit      int

	logger interfaces.Logger
}

func NewAgent(
	collector Collector,
	repository SenderRepository,
	pollInterval int,
	reportInterval int,
	rateLimit int,
	logger interfaces.Logger,
) *agent {
	return &agent{
		collector:      collector,
		repository:     repository,
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		rateLimit:      rateLimit,
		logger:         logger,
	}
}

func (a *agent) Run(ctx context.Context) {
	runtimeMetricCh := make(chan models.Metrics, 100)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		a.runtimeMetricUpdater(ctx, runtimeMetricCh)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		a.extendedMetricUpdater(ctx, runtimeMetricCh)
	}()

	for i := 0; i < a.rateLimit; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.senderSnapshot(ctx, runtimeMetricCh)
		}()
	}

	wg.Wait()
	close(runtimeMetricCh)
}

func (a *agent) senderSnapshot(ctx context.Context, metricCh chan models.Metrics) {
	metricStorage := make([]models.Metrics, 0)

	timer := time.NewTicker(time.Duration(a.reportInterval) * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			if len(metricStorage) == 0 {
				continue
			}
			a.sendMetrics(metricStorage)

			metricStorage = make([]models.Metrics, 0, len(metricStorage))
		case metric, ok := <-metricCh:
			if !ok {
				a.sendMetrics(metricStorage)

				return
			}

			metricStorage = append(metricStorage, metric)
		case <-ctx.Done():
			a.sendMetrics(metricStorage)

			return
		}
	}
}

func (a *agent) runtimeMetricUpdater(ctx context.Context, metricsCh chan models.Metrics) {
	counter := int64(0)

	timer := time.NewTicker(time.Duration(a.pollInterval) * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			metrics := a.collector.MetricsSnapshot(counter)
			counter++

			a.sendMetricToChannel(ctx, metricsCh, metrics)

		case <-ctx.Done():
			return
		}
	}
}

func (a *agent) extendedMetricUpdater(ctx context.Context, metricsCh chan models.Metrics) {
	timer := time.NewTicker(time.Duration(a.pollInterval) * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			metrics, err := a.collector.ExtendedMetricSnapshot()
			if err != nil {
				a.logger.Error("Ошибка сборка extended метрик", zap.Error(err))
			}

			a.sendMetricToChannel(ctx, metricsCh, metrics)
		case <-ctx.Done():
			return
		}
	}
}

func (a *agent) sendMetricToChannel(ctx context.Context, ch chan models.Metrics, metrics []models.Metrics) {
	for _, value := range metrics {
		select {
		case ch <- value:
		case <-ctx.Done():
			return
		}
	}
}

func (a *agent) sendMetrics(metrics []models.Metrics) {
	if len(metrics) == 0 {
		return
	}

	err := a.repository.SendBatchMetric(metrics)
	if err != nil {
		a.logger.Error("Ошибка отправки метрик", zap.Error(err))
	}
}
