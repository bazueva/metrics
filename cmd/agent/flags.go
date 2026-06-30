package main

import (
	"flag"
)

func parseFlags(config *config) {
	flag.Var(&config.metricServerAddr, "a", "Адрес сервера для отправки метрик")
	flag.DurationVar(&config.pollInterval, "p", pollInterval, "Частота опроса метрик")
	flag.DurationVar(&config.reportInterval, "r", reportInterval, "Частота отправки метрик на сервер")

	flag.Parse()
}
