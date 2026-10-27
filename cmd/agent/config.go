package main

import (
	"encoding/json"
	"flag"
	"os"

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
	agentConfig := config{
		MetricServerAddr: configpkg.ServerAddr{
			Host: "localhost",
			Port: 8080,
		},
	}

	providedFlags, err := parseFlags(&agentConfig)
	if err != nil {
		return agentConfig, err
	}

	if err = env.Parse(&agentConfig); err != nil {
		return agentConfig, err
	}

	if agentConfig.ConfigFile != "" {
		if err = readConfigFromFile(&agentConfig, providedFlags); err != nil {
			return agentConfig, err
		}
	}

	return agentConfig, nil
}

func parseFlags(config *config) (map[string]bool, error) {
	agentFlags := flag.NewFlagSet("", flag.ContinueOnError)

	agentFlags.Var(&config.MetricServerAddr, "a", "Адрес сервера для отправки метрик")
	agentFlags.IntVar(&config.PollInterval, "p", agent.PollInterval, "Частота опроса метрик")
	agentFlags.IntVar(&config.ReportInterval, "r", agent.ReportInterval, "Частота отправки метрик на сервер")
	agentFlags.StringVar(&config.SecretKey, "k", "", "Ключ для расчета hash")
	agentFlags.StringVar(&config.CryptoKey, "crypto-key", "", "Файл с публичным ключом")
	agentFlags.StringVar(&config.ConfigFile, "config", "", "Файл с конфигом")
	agentFlags.StringVar(&config.ConfigFile, "c", "", "Файл с конфигом")
	agentFlags.IntVar(&config.RateLimit, "l", 1, "Количество горутин")

	if err := agentFlags.Parse(os.Args[1:]); err != nil {
		return nil, err
	}

	providedFlags := make(map[string]bool)

	agentFlags.Visit(func(f *flag.Flag) {
		providedFlags[f.Name] = true
	})

	return providedFlags, nil
}

func readConfigFromFile(c *config, providedFlags map[string]bool) error {
	data, err := os.ReadFile(c.ConfigFile)
	if err != nil {
		return err
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	for _, field := range []struct {
		jsonName string
		flagName string
		envName  string
	}{
		{"address", "a", "ADDRESS"},
		{"report_interval", "r", "REPORT_INTERVAL"},
		{"poll_interval", "p", "POLL_INTERVAL"},
		{"key", "k", "KEY"},
		{"rate_limit", "l", "RATE_LIMIT"},
		{"crypto_key", "crypto-key", "CRYPTO_KEY"},
	} {
		if !canUseJSON(field.flagName, field.envName, providedFlags) {
			delete(fields, field.jsonName)
		}
	}

	filtered, err := json.Marshal(fields)
	if err != nil {
		return err
	}

	return json.Unmarshal(filtered, c)
}

func canUseJSON(flagName, envName string, providedFlags map[string]bool) bool {
	if providedFlags[flagName] {
		return false
	}

	if _, exists := os.LookupEnv(envName); exists {
		return false
	}

	return true
}
