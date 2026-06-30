package main

import (
	"flag"
)

func parseFlags(config *config) {
	flag.Var(&config.metricServerAddr, "a", "Адрес сервера для отправки метрик")

	flag.IntVar(&config.pollInterval, "p", pollInterval, "Частота опроса метрик")
	flag.IntVar(&config.reportInterval, "r", reportInterval, "Частота отправки метрик на сервер")

	flag.Parse()
}
