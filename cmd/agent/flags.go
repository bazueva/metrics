package main

import (
	"flag"

	"github.com/bazueva/metrics/internal/agent"
)

func parseFlags(config *config) {
	flag.Var(&config.metricServerAddr, "a", "Адрес сервера для отправки метрик")

	flag.IntVar(&config.pollInterval, "p", agent.PollInterval, "Частота опроса метрик")
	flag.IntVar(&config.reportInterval, "r", agent.ReportInterval, "Частота отправки метрик на сервер")

	flag.Parse()
}
