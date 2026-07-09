package main

import (
	"flag"
	"os"
	"strings"
	"time"

	configpkg "github.com/bazueva/metrics/cmd/config"
	"github.com/bazueva/metrics/internal/agent"
	"github.com/caarlos0/env/v11"
)

type config struct {
	MetricServerAddr configpkg.ServerAddr `env:"ADDRESS"`
	ReportInterval   time.Duration        `env:"REPORT_INTERVAL"`
	PollInterval     time.Duration        `env:"POLL_INTERVAL"`
}

func readConfig() (config, error) {
	agentConfig := config{
		MetricServerAddr: configpkg.ServerAddr{
			Host: "localhost",
			Port: 8080,
		},
	}

	err := parseFlags(&agentConfig)
	if err != nil {
		return agentConfig, err
	}

	err = env.Parse(&agentConfig)
	if err != nil {
		return agentConfig, err
	}

	return agentConfig, nil
}

func parseFlags(config *config) error {
	agentFlags := flag.NewFlagSet("", flag.ContinueOnError)
	agentFlags.Var(&config.MetricServerAddr, "a", "Адрес сервера для отправки метрик")

	agentFlags.DurationVar(&config.PollInterval, "p", agent.PollInterval, "Частота опроса метрик")
	agentFlags.DurationVar(&config.ReportInterval, "r", agent.ReportInterval, "Частота отправки метрик на сервер")

	if len(os.Args) > 1 {
		err := agentFlags.Parse(os.Args[1:])
		if err != nil && !strings.Contains(err.Error(), "flag provided but not defined:") {
			return err
		}
	}

	return nil
}
