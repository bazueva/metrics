package main

import (
	"os"
	"path/filepath"
	"testing"

	configpkg "github.com/bazueva/metrics/cmd/config"
	"github.com/bazueva/metrics/internal/agent"
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
		MetricServerAddr: configpkg.ServerAddr{
			Host: "localhost",
			Port: 8080,
		},
		ReportInterval: agent.ReportInterval,
		PollInterval:   agent.PollInterval,
		RateLimit:      1,
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
				"-p", "19",
				"-r", "48",
				"-k", "45666",
				"-l", "4",
				"-crypto-key", "flag.pem",
			},
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "local",
					Port: 1111,
				},
				ReportInterval: 48,
				PollInterval:   19,
				SecretKey:      "45666",
				RateLimit:      4,
				CryptoKey:      "flag.pem",
			},
		},
		{
			name: "with envs",
			envVars: map[string]string{
				"ADDRESS":         "env-host:8082",
				"REPORT_INTERVAL": "60",
				"POLL_INTERVAL":   "8",
				"KEY":             "env-secret",
				"RATE_LIMIT":      "4",
				"CRYPTO_KEY":      "env.pem",
			},
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "env-host",
					Port: 8082,
				},
				ReportInterval: 60,
				PollInterval:   8,
				SecretKey:      "env-secret",
				RateLimit:      4,
				CryptoKey:      "env.pem",
			},
		},
		{
			name: "with JSON config",
			configJSON: `{
				"address": "json-host:9090",
				"report_interval": 10,
				"poll_interval": 2,
				"key": "json-secret",
				"rate_limit": 5,
				"crypto_key": "json.pem"
			}`,
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "json-host",
					Port: 9090,
				},
				ReportInterval: 10,
				PollInterval:   2,
				SecretKey:      "json-secret",
				RateLimit:      5,
				CryptoKey:      "json.pem",
			},
		},
		{
			name:       "envs over flags over JSON",
			configFlag: "-config",
			configJSON: `{
				"address": "json-host:9090",
				"report_interval": 10,
				"poll_interval": 2,
				"key": "json-secret",
				"rate_limit": 5,
				"crypto_key": "json.pem"
			}`,
			args: []string{
				"-a", "flag-host:8081",
				"-r", "30",
				"-k", "flag-secret",
			},
			envVars: map[string]string{
				"ADDRESS":         "env-host:8082",
				"REPORT_INTERVAL": "60",
			},
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "env-host",
					Port: 8082,
				},
				ReportInterval: 60,
				PollInterval:   2,
				SecretKey:      "flag-secret",
				RateLimit:      5,
				CryptoKey:      "json.pem",
			},
		},
		{
			name:       "zero values filled from JSON",
			configFlag: "-c",
			configJSON: `{
				"report_interval": 10,
				"poll_interval": 2,
				"rate_limit": 5,
				"key": "json-secret"
			}`,
			args: []string{
				"-r", "0",
				"-p", "0",
				"-l", "0",
			},
			envVars: map[string]string{
				"KEY": "",
			},
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "localhost",
					Port: 8080,
				},
				ReportInterval: 10,
				PollInterval:   2,
				RateLimit:      5,
				SecretKey:      "json-secret",
			},
		},
		{
			name: "zero values use defaults",
			args: []string{
				"-r", "0",
				"-p", "0",
				"-l", "0",
			},
			want: defaultConfig,
		},
		{
			name:    "invalid flag value",
			args:    []string{"-p", "invalid"},
			wantErr: true,
		},
		{
			name: "invalid env value",
			envVars: map[string]string{
				"POLL_INTERVAL": "invalid",
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

			clearConfigEnv(t)

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

			for key, value := range tt.envVars {
				require.NoError(t, os.Setenv(key, value))
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

func Test_readConfigFromFile(t *testing.T) {
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
				"report_interval": 10,
				"poll_interval": 2,
				"key": "secret",
				"rate_limit": 5,
				"crypto_key": "public.pem"
			}`,
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "localhost",
					Port: 9090,
				},
				ReportInterval: 10,
				PollInterval:   2,
				SecretKey:      "secret",
				RateLimit:      5,
				CryptoKey:      "public.pem",
			},
		},
		{
			name:       "existing non-zero fields are preserved",
			configJSON: `{"report_interval": 10, "poll_interval": 2, "rate_limit": 5, "key": "json-secret"}`,
			initial: config{
				ReportInterval: 30,
				PollInterval:   7,
				RateLimit:      3,
				SecretKey:      "flag-secret",
			},
			want: config{
				ReportInterval: 30,
				PollInterval:   7,
				RateLimit:      3,
				SecretKey:      "flag-secret",
			},
		},
		{
			name:       "invalid JSON",
			configJSON: `{"report_interval": 10,}`,
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

func clearConfigEnv(t *testing.T) {
	t.Helper()

	envNames := []string{
		"ADDRESS",
		"REPORT_INTERVAL",
		"POLL_INTERVAL",
		"KEY",
		"RATE_LIMIT",
		"CRYPTO_KEY",
		"CONFIG",
	}

	for _, name := range envNames {
		value, exists := os.LookupEnv(name)

		require.NoError(t, os.Unsetenv(name))

		t.Cleanup(func() {
			if exists {
				_ = os.Setenv(name, value)
			} else {
				_ = os.Unsetenv(name)
			}
		})
	}
}
