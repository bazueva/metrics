package main

import (
	"flag"
	"os"

	configpkg "github.com/bazueva/metrics/cmd/config"
	"github.com/caarlos0/env/v11"
	"go.uber.org/zap"
)

const (
	storeIntervalDefault   = 300
	fileStoragePathDefault = "metrics.log"
)

type config struct {
	ServerAddr          configpkg.ServerAddr `env:"ADDRESS"`
	StoreInterval       int                  `env:"STORE_INTERVAL"`
	FileStoragePath     string               `env:"FILE_STORAGE_PATH"`
	LoadMetricsFromFile bool                 `env:"RESTORE"`

	logger *zap.Logger
}

func readConfig() (config, error) {
	cfg := config{
		ServerAddr: configpkg.ServerAddr{
			Host: "localhost",
			Port: 8080,
		},
	}

	err := parseFlags(&cfg)
	if err != nil {
		return config{}, err
	}

	err = env.Parse(&cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}

func parseFlags(config *config) error {
	serverFlags := flag.NewFlagSet("", flag.ContinueOnError)
	serverFlags.Var(&config.ServerAddr, "a", "address http server")
	serverFlags.IntVar(&config.StoreInterval, "i", storeIntervalDefault, "store interval in seconds")
	serverFlags.StringVar(&config.FileStoragePath, "f", fileStoragePathDefault, "Storage file path")
	serverFlags.BoolVar(&config.LoadMetricsFromFile, "r", false, "Load metrics from file")

	if len(os.Args) > 1 {
		err := serverFlags.Parse(os.Args[1:])
		if err != nil {
			return err
		}
	}

	return nil
}
