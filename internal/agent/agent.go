package agent

import (
	"fmt"
	"sync"
	"time"

	models "github.com/bazueva/metrics/internal/model"
)

const PollInterval = 2
const ReportInterval = 10

type Collector interface {
	MetricsSnapshot(counter int64) []models.Metrics
}

type SenderRepository interface {
	SendMetric(metrics models.Metrics) error
}

type agent struct {
	collector      Collector
	repository     SenderRepository
	metrics        []models.Metrics
	reportInterval int
	pollInterval   int

	mu sync.Mutex
}

func NewAgent(
	collector Collector,
	repository SenderRepository,
	pollInterval int,
	reportInterval int,
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
			time.Sleep(time.Duration(a.pollInterval) * time.Second)
		}
	}()

	for {
		time.Sleep(time.Duration(a.reportInterval) * time.Second)

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
		err = a.repository.SendMetric(value)
		if err != nil {
			return err
		}
	}

	return nil
}
