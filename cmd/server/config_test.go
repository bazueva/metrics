package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	configpkg "github.com/bazueva/metrics/cmd/config"
	"github.com/caarlos0/env/v11"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_readConfig(t *testing.T) {
	type test struct {
		name       string
		envVars    map[string]string
		args       []string
		configJSON string
		want       config
		wantErr    bool
	}

	tests := []test{
		{
			name: "empty envs and args",
			want: config{
				ServerAddr: configpkg.ServerAddr{
					Host: "localhost",
					Port: 8080,
				},
				StoreInterval:       300,
				FileStoragePath:     "metrics.log",
				LoadMetricsFromFile: false,
			},
		},
		{
			name: "with ADDRESS env",
			envVars: map[string]string{
				"ADDRESS": "test:6789",
			},
			want: config{
				ServerAddr: configpkg.ServerAddr{
					Host: "test",
					Port: 6789,
				},
				StoreInterval:       300,
				FileStoragePath:     "metrics.log",
				LoadMetricsFromFile: false,
			},
		},
		{
			name: "with args",
			args: []string{
				"cmd",
				"-a", "local:1111",
				"-i", "20",
				"-f", "f.log",
				"-r",
				"-k", "1222",
			},
			want: config{
				ServerAddr: configpkg.ServerAddr{
					Host: "local",
					Port: 1111,
				},
				StoreInterval:       20,
				FileStoragePath:     "f.log",
				LoadMetricsFromFile: true,
				SecretKey:           "1222",
			},
		},
		{
			name: "with args and env",
			envVars: map[string]string{
				"ADDRESS":           "test:8900",
				"STORE_INTERVAL":    "56",
				"FILE_STORAGE_PATH": "file_env.log",
				"RESTORE":           "true",
				"KEY":               "11111",
			},
			args: []string{
				"cmd",
				"-a", "local:1111",
				"-i", "20",
				"-f", "f.log",
				"-r", "false",
			},
			want: config{
				ServerAddr: configpkg.ServerAddr{
					Host: "test",
					Port: 8900,
				},
				StoreInterval:       56,
				FileStoragePath:     "file_env.log",
				LoadMetricsFromFile: true,
				SecretKey:           "11111",
			},
		},
		{
			name: "with JSON config",
			configJSON: `{
				"address": "json-host:9090",
				"store_interval": 10,
				"store_file": "json-metrics.log",
				"restore": true,
				"database_dsn": "postgres://localhost/metrics",
				"key": "json-secret",
				"audit_file": "audit.log",
				"audit_url": "http://localhost:8081",
				"pprof_port": 6060,
				"crypto_key": "private.pem"
			}`,
			want: config{
				ServerAddr: configpkg.ServerAddr{
					Host: "json-host",
					Port: 9090,
				},
				StoreInterval:       10,
				FileStoragePath:     "json-metrics.log",
				LoadMetricsFromFile: true,
				DatabaseDSN:         "postgres://localhost/metrics",
				SecretKey:           "json-secret",
				AuditFile:           "audit.log",
				AuditURL:            "http://localhost:8081",
				PprofPort:           6060,
				CryptoKey:           "private.pem",
			},
		},
		{
			name: "with JSON config and args",
			configJSON: `{
				"address": "json-host:9090",
				"store_interval": 10,
				"store_file": "json-metrics.log",
				"restore": true
			}`,
			args: []string{
				"cmd",
				"-a", "flag-host:8081",
				"-i", "30",
				"-r=false",
			},
			want: config{
				ServerAddr: configpkg.ServerAddr{
					Host: "flag-host",
					Port: 8081,
				},
				StoreInterval:       30,
				FileStoragePath:     "json-metrics.log",
				LoadMetricsFromFile: false,
			},
		},
		{
			name: "with JSON config and env",
			configJSON: `{
				"address": "json-host:9090",
				"store_interval": 10,
				"store_file": "json-metrics.log",
				"restore": true
			}`,
			envVars: map[string]string{
				"ADDRESS":        "env-host:8082",
				"STORE_INTERVAL": "60",
				"RESTORE":        "false",
			},
			want: config{
				ServerAddr: configpkg.ServerAddr{
					Host: "env-host",
					Port: 8082,
				},
				StoreInterval:       60,
				FileStoragePath:     "json-metrics.log",
				LoadMetricsFromFile: false,
			},
		},
		{
			name: "with JSON config, args and env",
			configJSON: `{
				"address": "json-host:9090",
				"store_interval": 10,
				"store_file": "json-metrics.log",
				"restore": true
			}`,
			args: []string{
				"cmd",
				"-a", "flag-host:8081",
				"-i", "30",
				"-f", "flag-metrics.log",
			},
			envVars: map[string]string{
				"ADDRESS":        "env-host:8082",
				"STORE_INTERVAL": "60",
			},
			want: config{
				ServerAddr: configpkg.ServerAddr{
					Host: "env-host",
					Port: 8082,
				},
				StoreInterval:       60,
				FileStoragePath:     "flag-metrics.log",
				LoadMetricsFromFile: true,
			},
		},
		{
			name: "invalid JSON config",
			configJSON: `{
		"address": "localhost:9090",
		"store_interval": 10,
	}`,
			wantErr: true,
		},
		{
			name: "invalid JSON field type",
			configJSON: `{
		"store_interval": "invalid"
	}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			t.Cleanup(func() {
				os.Args = oldArgs
			})

			oldEnv := os.Environ()
			t.Cleanup(func() {
				os.Clearenv()

				for _, env := range oldEnv {
					parts := strings.SplitN(env, "=", 2)
					if len(parts) == 2 {
						_ = os.Setenv(parts[0], parts[1])
					}
				}
			})

			os.Clearenv()

			for key, value := range tt.envVars {
				require.NoError(t, os.Setenv(key, value))
			}

			args := tt.args
			if len(args) == 0 {
				args = []string{"cmd"}
			}

			if tt.configJSON != "" {
				configPath := filepath.Join(t.TempDir(), "config.json")

				err := os.WriteFile(
					configPath,
					[]byte(tt.configJSON),
					0600,
				)
				require.NoError(t, err)

				tt.want.ConfigFile = configPath

				switch tt.name {
				case "with JSON config and args":
					args = append(args, "-c", configPath)

				case "with JSON config, args and env":
					args = append(args, "-config", configPath)

				default:
					require.NoError(t, os.Setenv("CONFIG", configPath))
				}
			}

			os.Args = args

			cfg, err := readConfig()

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, cfg)
		})
	}
}

func TestReadConfigFromFile(t *testing.T) {
	t.Run("success", func(t *testing.T) {

		configPath := filepath.Join(t.TempDir(), "config.json")

		data := []byte(`{
			"address": "localhost:9090",
			"store_interval": 10,
			"store_file": "test-metrics.log",
			"restore": true,
			"database_dsn": "postgres://localhost:5432/metrics",
			"key": "secret",
			"audit_file": "audit.log",
			"audit_url": "http://localhost:8081",
			"pprof_port": 6060,
			"crypto_key": "private.pem"
		}`)

		err := os.WriteFile(configPath, data, 0600)
		require.NoError(t, err)

		cfg := config{
			ConfigFile: configPath,
		}

		err = readConfigFromFile(&cfg, nil)
		require.NoError(t, err)

		require.Equal(t, "localhost", cfg.ServerAddr.Host)
		require.Equal(t, 9090, cfg.ServerAddr.Port)
		require.Equal(t, 10, cfg.StoreInterval)
		require.Equal(t, "test-metrics.log", cfg.FileStoragePath)
		require.True(t, cfg.LoadMetricsFromFile)
		require.Equal(t, "postgres://localhost:5432/metrics", cfg.DatabaseDSN)
		require.Equal(t, "secret", cfg.SecretKey)
		require.Equal(t, "audit.log", cfg.AuditFile)
		require.Equal(t, "http://localhost:8081", cfg.AuditURL)
		require.Equal(t, 6060, cfg.PprofPort)
		require.Equal(t, "private.pem", cfg.CryptoKey)
	})

	t.Run("flags have priority over JSON", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.json")

		data := []byte(`{
		"address": "localhost:9090",
		"store_interval": 10,
		"store_file": "test-metrics.log",
		"restore": true
	}`)

		err := os.WriteFile(configPath, data, 0600)
		require.NoError(t, err)

		cfg := config{
			ServerAddr: configpkg.ServerAddr{
				Host: "localhost",
				Port: 8081,
			},
			StoreInterval:       30,
			FileStoragePath:     "flags-metrics.log",
			LoadMetricsFromFile: false,
			ConfigFile:          configPath,
		}

		providedFlags := map[string]bool{
			"a": true,
			"i": true,
			"f": true,
			"r": true,
		}

		err = readConfigFromFile(&cfg, providedFlags)
		require.NoError(t, err)

		require.Equal(t, "localhost", cfg.ServerAddr.Host)
		require.Equal(t, 8081, cfg.ServerAddr.Port)
		require.Equal(t, 30, cfg.StoreInterval)
		require.Equal(t, "flags-metrics.log", cfg.FileStoragePath)
		require.False(t, cfg.LoadMetricsFromFile)
	})

	t.Run("env have priority over JSON", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.json")

		data := []byte(`{
		"address": "localhost:9090",
		"store_interval": 10,
		"store_file": "test-metrics.log",
		"restore": true
	}`)

		err := os.WriteFile(configPath, data, 0600)
		require.NoError(t, err)

		t.Setenv("ADDRESS", "localhost:8081")
		t.Setenv("STORE_INTERVAL", "30")
		t.Setenv("FILE_STORAGE_PATH", "env-metrics.log")
		t.Setenv("RESTORE", "false")

		cfg := config{
			ConfigFile: configPath,
		}

		err = env.Parse(&cfg)
		require.NoError(t, err)

		err = readConfigFromFile(&cfg, nil)
		require.NoError(t, err)

		require.Equal(t, "localhost", cfg.ServerAddr.Host)
		require.Equal(t, 8081, cfg.ServerAddr.Port)
		require.Equal(t, 30, cfg.StoreInterval)
		require.Equal(t, "env-metrics.log", cfg.FileStoragePath)
		require.False(t, cfg.LoadMetricsFromFile)
	})

	t.Run("file does not exist", func(t *testing.T) {
		cfg := config{
			ConfigFile: filepath.Join(t.TempDir(), "missing.json"),
		}

		err := readConfigFromFile(&cfg, nil)

		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.json")

		data := []byte(`{
		"address": "localhost:9090",
		"store_interval": 10,
	}`)

		err := os.WriteFile(configPath, data, 0600)
		require.NoError(t, err)

		cfg := config{
			ConfigFile: configPath,
		}

		err = readConfigFromFile(&cfg, nil)

		require.Error(t, err)
	})

	t.Run("invalid field type", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.json")

		data := []byte(`{
		"store_interval": "invalid"
	}`)

		err := os.WriteFile(configPath, data, 0600)
		require.NoError(t, err)

		cfg := config{
			ConfigFile: configPath,
		}

		err = readConfigFromFile(&cfg, nil)

		require.Error(t, err)
	})

	t.Run("missing JSON fields preserve defaults", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.json")

		data := []byte(`{
		"store_interval": 10
	}`)

		err := os.WriteFile(configPath, data, 0600)
		require.NoError(t, err)

		cfg := config{
			ServerAddr: configpkg.ServerAddr{
				Host: "localhost",
				Port: 8080,
			},
			StoreInterval:       storeIntervalDefault,
			FileStoragePath:     fileStoragePathDefault,
			LoadMetricsFromFile: false,
			ConfigFile:          configPath,
		}

		err = readConfigFromFile(&cfg, nil)
		require.NoError(t, err)

		require.Equal(t, 10, cfg.StoreInterval)
		require.Equal(t, "localhost", cfg.ServerAddr.Host)
		require.Equal(t, 8080, cfg.ServerAddr.Port)
		require.Equal(t, fileStoragePathDefault, cfg.FileStoragePath)
		require.False(t, cfg.LoadMetricsFromFile)
	})
}
