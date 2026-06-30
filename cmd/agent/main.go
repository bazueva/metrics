package main

import (
	"fmt"
	"time"

	config2 "github.com/bazueva/metrics/cmd/config"
	"github.com/bazueva/metrics/internal/agent"
	models "github.com/bazueva/metrics/internal/model"
	"github.com/bazueva/metrics/internal/repository"
)

const pollInterval = 2
const reportInterval = 10

type config struct {
	metricServerAddr config2.ServerAddr
	reportInterval   int
	pollInterval     int
}

func readConfig() config {
	agentConfig := config{
		metricServerAddr: config2.ServerAddr{
			Host: "localhost",
			Port: 8080,
		},
	}

	parseFlags(&agentConfig)

	return agentConfig
}

func main() {
	agentConfig := readConfig()

	metricRepository, err := repository.NewRepository(fmt.Sprintf("http://%s", agentConfig.metricServerAddr.String()))
	if err != nil {
		panic(err)
	}

	metricSender := agent.NewSender(metricRepository)

	counter := int64(0)
	collector := agent.NewCollector()

	var metrics []models.Metrics

	go func() {
		for {
			metrics = collector.MetricsSnapshot(counter)
			counter++
			time.Sleep(time.Duration(agentConfig.pollInterval) * time.Second)
		}
	}()

	for {
		time.Sleep(time.Duration(agentConfig.reportInterval) * time.Second)

		err := metricSender.SendSnapshot(metrics)
		if err != nil {
			fmt.Println(err)
		}
	}
}
