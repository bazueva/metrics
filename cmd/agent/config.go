package main

import (
	"encoding/json"
	"flag"
	"os"

	"dario.cat/mergo"
	configpkg "github.com/bazueva/metrics/cmd/config"
	"github.com/bazueva/metrics/internal/agent"
	"github.com/caarlos0/env/v11"
)

type config struct {
	MetricServerAddr configpkg.ServerAddr `env:"ADDRESS" json:"address"`
	ReportInterval   int                  `env:"REPORT_INTERVAL" json:"report_interval"`
	PollInterval     int                  `env:"POLL_INTERVAL" json:"poll_interval"`
	SecretKey        string               `env:"KEY" json:"key"`
	RateLimit        int                  `env:"RATE_LIMIT" json:"rate_limit"`
	CryptoKey        string               `env:"CRYPTO_KEY" json:"crypto_key"`
	ConfigFile       string               `env:"CONFIG" json:"-"`
}

func readConfig() (config, error) {
	var agentConfig config

	if err := parseFlags(&agentConfig); err != nil {
		return config{}, err
	}

	if err := env.Parse(&agentConfig); err != nil {
		return config{}, err
	}

	if agentConfig.ConfigFile != "" {
		if err := readConfigFromFile(&agentConfig); err != nil {
			return config{}, err
		}
	}

	defaultConfig := config{
		MetricServerAddr: configpkg.ServerAddr{
			Host: "localhost",
			Port: 8080,
		},
		PollInterval:   agent.PollInterval,
		ReportInterval: agent.ReportInterval,
		RateLimit:      1,
	}

	if err := mergo.Merge(&agentConfig, defaultConfig); err != nil {
		return config{}, err
	}

	return agentConfig, nil
}

func parseFlags(config *config) error {
	agentFlags := flag.NewFlagSet("", flag.ContinueOnError)

	agentFlags.Var(&config.MetricServerAddr, "a", "Адрес сервера для отправки метрик")
	agentFlags.IntVar(&config.PollInterval, "p", config.PollInterval, "Частота опроса метрик")
	agentFlags.IntVar(&config.ReportInterval, "r", config.ReportInterval, "Частота отправки метрик на сервер")
	agentFlags.StringVar(&config.SecretKey, "k", "", "Ключ для расчета hash")
	agentFlags.StringVar(&config.CryptoKey, "crypto-key", "", "Файл с публичным ключом")
	agentFlags.StringVar(&config.ConfigFile, "config", "", "Файл с конфигом")
	agentFlags.StringVar(&config.ConfigFile, "c", "", "Файл с конфигом")
	agentFlags.IntVar(&config.RateLimit, "l", config.RateLimit, "Количество горутин")

	return agentFlags.Parse(os.Args[1:])
}

func readConfigFromFile(c *config) error {
	data, err := os.ReadFile(c.ConfigFile)
	if err != nil {
		return err
	}

	var fileConfig config
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return err
	}

	return mergo.Merge(c, fileConfig)
}
