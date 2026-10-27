package main

import (
	"encoding/json"
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
	ConfigFile          string               `env:"CONFIG" json:"config"`

	logger *zap.Logger
}

func readConfig() (config, error) {
	cfg := config{
		ServerAddr: configpkg.ServerAddr{
			Host: "localhost",
			Port: 8080,
		},
	}

	providedFlags, err := parseFlags(&cfg)
	if err != nil {
		return config{}, err
	}

	if err = env.Parse(&cfg); err != nil {
		return config{}, err
	}

	if cfg.ConfigFile != "" {
		if err = readConfigFromFile(&cfg, providedFlags); err != nil {
			return config{}, err
		}
	}

	return cfg, nil
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
		{"store_interval", "i", "STORE_INTERVAL"},
		{"store_file", "f", "FILE_STORAGE_PATH"},
		{"restore", "r", "RESTORE"},
		{"database_dsn", "d", "DATABASE_DSN"},
		{"key", "k", "KEY"},
		{"audit_file", "audit-file", "AUDIT_FILE"},
		{"audit_url", "audit-url", "AUDIT_URL"},
		{"pprof_port", "pprof-port", "PPROF_PORT"},
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
func parseFlags(config *config) (map[string]bool, error) {
	serverFlags := flag.NewFlagSet("", flag.ContinueOnError)

	serverFlags.Var(&config.ServerAddr, "a", "address http server")
	serverFlags.IntVar(&config.StoreInterval, "i", storeIntervalDefault, "store interval in seconds")
	serverFlags.StringVar(&config.FileStoragePath, "f", fileStoragePathDefault, "Storage file path")
	serverFlags.BoolVar(&config.LoadMetricsFromFile, "r", false, "Load metrics from file")
	serverFlags.StringVar(&config.DatabaseDSN, "d", "", "Database DSN")
	serverFlags.StringVar(&config.SecretKey, "k", "", "Ключ для расчета hash")
	serverFlags.StringVar(&config.AuditFile, "audit-file", "", "Audit file path")
	serverFlags.StringVar(&config.AuditURL, "audit-url", "", "Audit url")
	serverFlags.StringVar(&config.CryptoKey, "crypto-key", "", "Файл с приватным ключом")
	serverFlags.StringVar(&config.ConfigFile, "config", "", "Файл с конфигом")
	serverFlags.StringVar(&config.ConfigFile, "c", "", "Файл с конфигом")
	serverFlags.IntVar(&config.PprofPort, "pprof-port", 0, "Pprof port")

	if err := serverFlags.Parse(os.Args[1:]); err != nil {
		return nil, err
	}

	providedFlags := make(map[string]bool)

	serverFlags.Visit(func(f *flag.Flag) {
		providedFlags[f.Name] = true
	})

	return providedFlags, nil
}

func canUseJSON(
	flagName string,
	envName string,
	providedFlags map[string]bool,
) bool {
	if providedFlags[flagName] {
		return false
	}

	if _, exists := os.LookupEnv(envName); exists {
		return false
	}

	return true
}
