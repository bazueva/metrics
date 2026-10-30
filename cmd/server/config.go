package main

import (
	"encoding/json"
	"flag"
	"os"

	"dario.cat/mergo"
	configpkg "github.com/bazueva/metrics/cmd/config"
	"github.com/caarlos0/env/v11"
	"go.uber.org/zap"
)

const (
	storeIntervalDefault   = 300
	fileStoragePathDefault = "metrics.log"
)

type config struct {
	ServerAddr          configpkg.ServerAddr `env:"ADDRESS" json:"address"`
	StoreInterval       int                  `env:"STORE_INTERVAL" json:"store_interval"`
	FileStoragePath     string               `env:"FILE_STORAGE_PATH" json:"store_file"`
	LoadMetricsFromFile bool                 `env:"RESTORE" json:"restore"`
	DatabaseDSN         string               `env:"DATABASE_DSN" json:"database_dsn"`
	SecretKey           string               `env:"KEY" json:"key"`
	AuditFile           string               `env:"AUDIT_FILE" json:"audit_file"`
	AuditURL            string               `env:"AUDIT_URL" json:"audit_url"`
	PprofPort           int                  `env:"PPROF_PORT" json:"pprof_port"`
	CryptoKey           string               `env:"CRYPTO_KEY" json:"crypto_key"`
	ConfigFile          string               `env:"CONFIG" json:"-"`

	logger *zap.Logger
}

func readConfig() (config, error) {
	var cfg config

	if err := parseFlags(&cfg); err != nil {
		return config{}, err
	}

	if err := env.Parse(&cfg); err != nil {
		return config{}, err
	}

	if cfg.ConfigFile != "" {
		if err := readConfigFromFile(&cfg); err != nil {
			return config{}, err
		}
	}

	defaultConfig := config{
		ServerAddr: configpkg.ServerAddr{
			Host: "localhost",
			Port: 8080,
		},
		StoreInterval:   storeIntervalDefault,
		FileStoragePath: fileStoragePathDefault,
	}

	if err := mergo.Merge(&cfg, defaultConfig); err != nil {
		return config{}, err
	}

	return cfg, nil
}

func readConfigFromFile(cfg *config) error {
	data, err := os.ReadFile(cfg.ConfigFile)
	if err != nil {
		return err
	}

	var fileConfig config
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return err
	}

	return mergo.Merge(cfg, fileConfig)
}

func parseFlags(config *config) error {
	serverFlags := flag.NewFlagSet("", flag.ContinueOnError)

	serverFlags.Var(&config.ServerAddr, "a", "address http server")
	serverFlags.IntVar(&config.StoreInterval, "i", config.StoreInterval, "store interval in seconds")
	serverFlags.StringVar(&config.FileStoragePath, "f", config.FileStoragePath, "Storage file path")
	serverFlags.BoolVar(&config.LoadMetricsFromFile, "r", false, "Load metrics from file")
	serverFlags.StringVar(&config.DatabaseDSN, "d", "", "Database DSN")
	serverFlags.StringVar(&config.SecretKey, "k", "", "Ключ для расчета hash")
	serverFlags.StringVar(&config.AuditFile, "audit-file", "", "Audit file path")
	serverFlags.StringVar(&config.AuditURL, "audit-url", "", "Audit url")
	serverFlags.StringVar(&config.CryptoKey, "crypto-key", "", "Файл с приватным ключом")
	serverFlags.StringVar(&config.ConfigFile, "config", "", "Файл с конфигом")
	serverFlags.StringVar(&config.ConfigFile, "c", "", "Файл с конфигом")
	serverFlags.IntVar(&config.PprofPort, "pprof-port", 0, "Pprof port")

	return serverFlags.Parse(os.Args[1:])
}
