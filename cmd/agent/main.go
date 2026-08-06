package main

import (
	"fmt"
	"log"

	"github.com/bazueva/metrics/internal/agent"
	"github.com/bazueva/metrics/internal/agent/collector"
	"github.com/bazueva/metrics/internal/repository/metric"
	"go.uber.org/zap"
)

func main() {
	agentConfig, err := readConfig()
	if err != nil {
		panic(err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	defer logger.Sync()

	metricRepository, err := metric.NewRepository(
		fmt.Sprintf("http://%s", agentConfig.MetricServerAddr.String()),
		agentConfig.SecretKey,
		logger,
	)
	if err != nil {
		log.Fatal(err)
	}

	metricsAgent := agent.NewAgent(
		collector.NewCollector(),
		metricRepository,
		agentConfig.PollInterval,
		agentConfig.ReportInterval,
	)
	metricsAgent.Run()
}
