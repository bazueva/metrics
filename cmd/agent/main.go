package main

import (
	"context"
	"crypto/rsa"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bazueva/metrics/internal/agent"
	"github.com/bazueva/metrics/internal/agent/collector"
	"github.com/bazueva/metrics/internal/buildinfo"
	"github.com/bazueva/metrics/internal/crypto"
	"github.com/bazueva/metrics/internal/repository/metric"
	"go.uber.org/zap"
)

var buildVersion string
var buildDate string
var buildCommit string

func main() {
	buildinfo.Print(buildVersion, buildDate, buildCommit)

	agentConfig, err := readConfig()
	if err != nil {
		panic(err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	defer logger.Sync()

	var publicKey *rsa.PublicKey

	if agentConfig.CryptoKey != "" {
		publicKey, err = crypto.LoadPublicKey(agentConfig.CryptoKey)
		if err != nil {
			log.Fatal(err)
		}
	}

	metricRepository, err := metric.NewRepository(
		fmt.Sprintf("http://%s", agentConfig.MetricServerAddr.String()),
		agentConfig.SecretKey,
		logger,
		publicKey,
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
		logger.Info("Получен Ctrl+C, останавливаемся...")
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
