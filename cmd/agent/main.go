package main

import (
	"fmt"

	configpkg "github.com/bazueva/metrics/cmd/config"
	"github.com/bazueva/metrics/internal/agent"
	"github.com/bazueva/metrics/internal/agent/collector"
	"github.com/bazueva/metrics/internal/repository"
)

type config struct {
	metricServerAddr configpkg.ServerAddr
	reportInterval   int
	pollInterval     int
}

func readConfig() config {
	agentConfig := config{
		metricServerAddr: configpkg.ServerAddr{
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

	metricsAgent := agent.NewAgent(collector.NewCollector(), metricRepository, agentConfig.pollInterval, agentConfig.reportInterval)
	metricsAgent.Run()
}
