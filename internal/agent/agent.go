package agent

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	models "github.com/bazueva/metrics/internal/model"
)

const PollInterval = time.Second * 2
const ReportInterval = time.Second * 10

type Collector interface {
	MetricsSnapshot(counter int64) []models.Metrics
}

type SenderRepository interface {
	SendMetric(metricType string, metricName string, metricValue string) error
}

type agent struct {
	collector      Collector
	repository     SenderRepository
	metrics        []models.Metrics
	reportInterval time.Duration
	pollInterval   time.Duration

	mu sync.Mutex
}

func NewAgent(
	collector Collector,
	repository SenderRepository,
	pollInterval time.Duration,
	reportInterval time.Duration,
) *agent {
	return &agent{
		collector:      collector,
		repository:     repository,
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
	}
}

func (a *agent) Run() {
	counter := int64(0)

	go func() {
		for {
			a.updateMetric(counter)
			counter++
			time.Sleep(a.pollInterval)
		}
	}()

	for {
		time.Sleep(a.reportInterval)

		err := a.sendSnapshot()
		if err != nil {
			fmt.Println(err)
		}
	}
}

func (a *agent) updateMetric(counter int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.metrics = a.collector.MetricsSnapshot(counter)
}

func (a *agent) sendSnapshot() error {
	a.mu.Lock()
	metrics := a.metrics
	a.metrics = nil
	a.mu.Unlock()

	var err error
	for _, value := range metrics {
		var metricValue string
		if value.MType == models.Counter {
			metricValue = strconv.FormatInt(*value.Delta, 10)
		} else {
			metricValue = strconv.FormatFloat(*value.Value, 'f', -1, 64)
		}

		err = a.repository.SendMetric(value.MType, value.ID, metricValue)
		if err != nil {
			return err
		}
	}

	return nil
}
