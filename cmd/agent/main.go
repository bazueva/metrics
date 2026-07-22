package main

import (
	"fmt"

	"github.com/bazueva/metrics/internal/agent"
	"github.com/bazueva/metrics/internal/agent/collector"
	"github.com/bazueva/metrics/internal/repository/metric"
)

func main() {
	agentConfig, err := readConfig()
	if err != nil {
		panic(err)
	}

	metricRepository, err := metric.NewRepository(fmt.Sprintf("http://%s", agentConfig.MetricServerAddr.String()))
	if err != nil {
		panic(err)
	}

	metricsAgent := agent.NewAgent(collector.NewCollector(), metricRepository, agentConfig.PollInterval, agentConfig.ReportInterval)
	metricsAgent.Run()
}
