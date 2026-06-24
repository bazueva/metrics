package main

import (
	"fmt"
	"time"

	"github.com/bazueva/metrics/internal/agent"
	models "github.com/bazueva/metrics/internal/model"
	"github.com/bazueva/metrics/internal/repository"
)

const pollInterval = 2 * time.Second
const reportInterval = 10 * time.Second

func main() {
	counter := int64(0)
	collector := agent.NewCollector()
	metricSender := agent.NewSender(repository.NewRepository())

	var metrics []models.Metrics

	go func() {
		for {
			metrics = collector.MetricsSnapshot(counter)
			counter++
			time.Sleep(pollInterval)
		}
	}()

	for {
		err := metricSender.SendSnapshot(metrics)
		if err != nil {
			fmt.Println(err)
		}

		time.Sleep(reportInterval)
	}
}
