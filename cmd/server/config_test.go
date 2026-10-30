package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	configpkg "github.com/bazueva/metrics/cmd/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_readConfig(t *testing.T) {
	type test struct {
		name       string
		envVars    map[string]string
		args       []string
		configJSON string
		configFlag string
		want       config
		wantErr    bool
	}

	defaultConfig := config{
		ServerAddr: configpkg.ServerAddr{
			Host: "localhost",
			Port: 8080,
		},
		StoreInterval:   storeIntervalDefault,
		FileStoragePath: fileStoragePathDefault,
	}

	tests := []test{
		{
			name: "empty envs and args",
			want: defaultConfig,
		},
		{
			name: "with args",
			args: []string{
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
			name: "with envs",
			envVars: map[string]string{
				"ADDRESS":           "env-host:8082",
				"STORE_INTERVAL":    "60",
				"FILE_STORAGE_PATH": "env-metrics.log",
				"RESTORE":           "true",
				"KEY":               "env-secret",
			},
			want: config{
				ServerAddr: configpkg.ServerAddr{
					Host: "env-host",
					Port: 8082,
				},
				StoreInterval:       60,
				FileStoragePath:     "env-metrics.log",
				LoadMetricsFromFile: true,
				SecretKey:           "env-secret",
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
			name:       "envs over flags over JSON",
			configFlag: "-config",
			configJSON: `{
				"address": "json-host:9090",
				"store_interval": 10,
				"store_file": "json-metrics.log",
				"restore": true,
				"key": "json-secret"
			}`,
			args: []string{
				"-a", "flag-host:8081",
				"-i", "30",
				"-f", "flag-metrics.log",
				"-k", "flag-secret",
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
				SecretKey:           "flag-secret",
			},
		},
		{
			name:       "explicit false flag is replaced by JSON true",
			configJSON: `{"restore": true}`,
			args:       []string{"-r=false"},
			want: config{
				ServerAddr: configpkg.ServerAddr{
					Host: "localhost",
					Port: 8080,
				},
				StoreInterval:       storeIntervalDefault,
				FileStoragePath:     fileStoragePathDefault,
				LoadMetricsFromFile: true,
			},
		},
		{
			name: "zero and empty values get defaults",
			args: []string{"-i=0", "-f="},
			want: defaultConfig,
		},
		{
			name:    "invalid flag value",
			args:    []string{"-i", "invalid"},
			wantErr: true,
		},
		{
			name: "invalid env value",
			envVars: map[string]string{
				"STORE_INTERVAL": "invalid",
			},
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

			args := append([]string{"cmd"}, tt.args...)

			if tt.configJSON != "" {
				configPath := filepath.Join(t.TempDir(), "config.json")

				require.NoError(t, os.WriteFile(
					configPath,
					[]byte(tt.configJSON),
					0600,
				))

				tt.want.ConfigFile = configPath

				if tt.configFlag != "" {
					args = append(args, tt.configFlag, configPath)
				} else {
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
	tests := []struct {
		name        string
		configJSON  string
		initial     config
		want        config
		wantErr     bool
		missingFile bool
	}{
		{
			name: "success",
			configJSON: `{
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
			}`,
			want: config{
				ServerAddr: configpkg.ServerAddr{
					Host: "localhost",
					Port: 9090,
				},
				StoreInterval:       10,
				FileStoragePath:     "test-metrics.log",
				LoadMetricsFromFile: true,
				DatabaseDSN:         "postgres://localhost:5432/metrics",
				SecretKey:           "secret",
				AuditFile:           "audit.log",
				AuditURL:            "http://localhost:8081",
				PprofPort:           6060,
				CryptoKey:           "private.pem",
			},
		},
		{
			name:       "existing non-zero fields are not overwritten",
			configJSON: `{"store_interval": 10, "restore": true}`,
			initial: config{
				ServerAddr: configpkg.ServerAddr{
					Host: "localhost",
					Port: 8080,
				},
				StoreInterval:   storeIntervalDefault,
				FileStoragePath: fileStoragePathDefault,
			},
			want: config{
				ServerAddr: configpkg.ServerAddr{
					Host: "localhost",
					Port: 8080,
				},
				StoreInterval:       storeIntervalDefault,
				FileStoragePath:     fileStoragePathDefault,
				LoadMetricsFromFile: true,
			},
		},
		{
			name:       "invalid JSON",
			configJSON: `{"store_interval": 10,}`,
			wantErr:    true,
		},
		{
			name:        "file does not exist",
			missingFile: true,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			configPath := filepath.Join(t.TempDir(), "config.json")

			if !tt.missingFile {
				require.NoError(t, os.WriteFile(
					configPath,
					[]byte(tt.configJSON),
					0600,
				))
			}

			cfg := tt.initial
			cfg.ConfigFile = configPath

			err := readConfigFromFile(&cfg)

			if tt.missingFile {
				require.ErrorIs(t, err, os.ErrNotExist)
				return
			}

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			tt.want.ConfigFile = configPath
			assert.Equal(t, tt.want, cfg)
		})
	}
}
