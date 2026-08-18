package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("Получен Ctrl+C, останавливаемся...")
		cancel()
	}()

	metricsAgent := agent.NewAgent(
		collector.NewCollector(),
		metricRepository,
		agentConfig.PollInterval,
		agentConfig.ReportInterval,
		agentConfig.RateLimit,
		logger,
	)
	metricsAgent.Run(ctx)
}
