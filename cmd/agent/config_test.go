package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	configpkg "github.com/bazueva/metrics/cmd/config"
	"github.com/bazueva/metrics/internal/agent"
	"github.com/caarlos0/env/v11"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_readConfig(t *testing.T) {
	type test struct {
		name        string
		envVars     map[string]string
		args        []string
		configJSON  string
		configFlag  string
		missingFile bool
		want        config
		wantErr     bool
	}

	tests := []test{
		{
			name: "empty envs and args",
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "localhost",
					Port: 8080,
				},
				ReportInterval: agent.ReportInterval,
				PollInterval:   agent.PollInterval,
				RateLimit:      1,
			},
		},
		{
			name: "with ADDRESS env",
			envVars: map[string]string{
				"ADDRESS": "test:6789",
			},
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "test",
					Port: 6789,
				},
				ReportInterval: agent.ReportInterval,
				PollInterval:   agent.PollInterval,
				RateLimit:      1,
			},
		},
		{
			name: "with REPORT_INTERVAL env",
			envVars: map[string]string{
				"REPORT_INTERVAL": "2",
			},
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "localhost",
					Port: 8080,
				},
				ReportInterval: 2,
				PollInterval:   agent.PollInterval,
				RateLimit:      1,
			},
		},
		{
			name: "with POLL_INTERVAL env",
			envVars: map[string]string{
				"POLL_INTERVAL": "5",
			},
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "localhost",
					Port: 8080,
				},
				ReportInterval: agent.ReportInterval,
				PollInterval:   5,
				RateLimit:      1,
			},
		},
		{
			name: "with all envs",
			envVars: map[string]string{
				"POLL_INTERVAL":   "5",
				"REPORT_INTERVAL": "5",
				"ADDRESS":         "test:8900",
				"RATE_LIMIT":      "45",
			},
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "test",
					Port: 8900,
				},
				ReportInterval: 5,
				PollInterval:   5,
				RateLimit:      45,
			},
		},
		{
			name: "with args",
			args: []string{
				"-a", "local:1111",
				"-p", "19",
				"-r", "48",
				"-k", "45666",
				"-l", "4",
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
			},
		},
		{
			name: "with args and envs",
			envVars: map[string]string{
				"POLL_INTERVAL":   "5",
				"REPORT_INTERVAL": "5",
				"ADDRESS":         "test:8900",
				"KEY":             "5111",
				"RATE_LIMIT":      "45",
			},
			args: []string{
				"-a", "local:1111",
				"-p", "19",
				"-r", "48",
				"-k", "45666",
				"-l", "4",
			},
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "test",
					Port: 8900,
				},
				ReportInterval: 5,
				PollInterval:   5,
				SecretKey:      "5111",
				RateLimit:      45,
			},
		},
		{
			name: "with JSON from CONFIG env",
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
			name:       "with JSON from -c flag",
			configFlag: "-c",
			configJSON: `{
				"address": "json-host:9090",
				"report_interval": 10,
				"poll_interval": 2,
				"rate_limit": 5
			}`,
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "json-host",
					Port: 9090,
				},
				ReportInterval: 10,
				PollInterval:   2,
				RateLimit:      5,
			},
		},
		{
			name:       "with JSON from -config flag",
			configFlag: "-config",
			configJSON: `{
				"address": "json-host:9090",
				"report_interval": 10,
				"poll_interval": 2,
				"rate_limit": 5
			}`,
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "json-host",
					Port: 9090,
				},
				ReportInterval: 10,
				PollInterval:   2,
				RateLimit:      5,
			},
		},
		{
			name:       "flags have priority over JSON",
			configFlag: "-c",
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
				"-p", "7",
				"-k", "flag-secret",
				"-l", "3",
				"-crypto-key", "flag.pem",
			},
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "flag-host",
					Port: 8081,
				},
				ReportInterval: 30,
				PollInterval:   7,
				SecretKey:      "flag-secret",
				RateLimit:      3,
				CryptoKey:      "flag.pem",
			},
		},
		{
			name: "envs have priority over JSON",
			configJSON: `{
				"address": "json-host:9090",
				"report_interval": 10,
				"poll_interval": 2,
				"key": "json-secret",
				"rate_limit": 5,
				"crypto_key": "json.pem"
			}`,
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
			name:       "explicit zero flag over JSON",
			configFlag: "-c",
			configJSON: `{
				"rate_limit": 5
			}`,
			args: []string{
				"-l", "0",
			},
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "localhost",
					Port: 8080,
				},
				ReportInterval: agent.ReportInterval,
				PollInterval:   agent.PollInterval,
				RateLimit:      0,
			},
		},
		{
			name: "missing JSON fields preserve defaults",
			configJSON: `{
				"report_interval": 10
			}`,
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "localhost",
					Port: 8080,
				},
				ReportInterval: 10,
				PollInterval:   agent.PollInterval,
				RateLimit:      1,
			},
		},
		{
			name:       "invalid JSON",
			configJSON: `{"report_interval": 10,}`,
			wantErr:    true,
		},
		{
			name:       "invalid JSON field type",
			configJSON: `{"poll_interval": "invalid"}`,
			wantErr:    true,
		},
		{
			name:       "invalid JSON address",
			configJSON: `{"address": "invalid-address"}`,
			wantErr:    true,
		},
		{
			name:        "config file does not exist",
			configFlag:  "-c",
			missingFile: true,
			wantErr:     true,
		},
		{
			name: "invalid env value",
			envVars: map[string]string{
				"POLL_INTERVAL": "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid flag value",
			args: []string{
				"-p", "invalid",
			},
			wantErr: true,
		},
		{
			name: "unknown flag",
			args: []string{
				"-unknown",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() {
				os.Args = oldArgs
			}()

			oldEnv := os.Environ()
			os.Clearenv()

			defer func() {
				os.Clearenv()

				for _, env := range oldEnv {
					parts := strings.SplitN(env, "=", 2)
					_ = os.Setenv(parts[0], parts[1])
				}
			}()

			args := append([]string{"cmd"}, tt.args...)

			if tt.configJSON != "" || tt.missingFile {
				configPath := filepath.Join(t.TempDir(), "config.json")

				if !tt.missingFile {
					err := os.WriteFile(
						configPath,
						[]byte(tt.configJSON),
						0600,
					)
					require.NoError(t, err)
				}

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
	type test struct {
		name          string
		configJSON    string
		config        config
		providedFlags map[string]bool
		envVars       map[string]string
		want          config
		wantErr       bool
		missingFile   bool
	}

	tests := []test{
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
			name: "flags have priority over JSON",
			configJSON: `{
				"address": "json-host:9090",
				"report_interval": 10,
				"poll_interval": 2,
				"key": "json-secret",
				"rate_limit": 5,
				"crypto_key": "json.pem"
			}`,
			config: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "flag-host",
					Port: 8081,
				},
				ReportInterval: 30,
				PollInterval:   7,
				SecretKey:      "flag-secret",
				RateLimit:      3,
				CryptoKey:      "flag.pem",
			},
			providedFlags: map[string]bool{
				"a":          true,
				"r":          true,
				"p":          true,
				"k":          true,
				"l":          true,
				"crypto-key": true,
			},
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "flag-host",
					Port: 8081,
				},
				ReportInterval: 30,
				PollInterval:   7,
				SecretKey:      "flag-secret",
				RateLimit:      3,
				CryptoKey:      "flag.pem",
			},
		},
		{
			name: "env have priority over JSON",
			configJSON: `{
				"address": "json-host:9090",
				"report_interval": 10,
				"poll_interval": 2,
				"key": "json-secret",
				"rate_limit": 5,
				"crypto_key": "json.pem"
			}`,
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
			name: "missing JSON fields preserve defaults",
			configJSON: `{
				"report_interval": 10
			}`,
			config: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "localhost",
					Port: 8080,
				},
				ReportInterval: agent.ReportInterval,
				PollInterval:   agent.PollInterval,
				RateLimit:      1,
			},
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "localhost",
					Port: 8080,
				},
				ReportInterval: 10,
				PollInterval:   agent.PollInterval,
				RateLimit:      1,
			},
		},
		{
			name:       "invalid JSON",
			configJSON: `{"report_interval": 10,}`,
			wantErr:    true,
		},
		{
			name:       "invalid field type",
			configJSON: `{"poll_interval": "invalid"}`,
			wantErr:    true,
		},
		{
			name:       "invalid address",
			configJSON: `{"address": "invalid-address"}`,
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
			clearConfigEnv(t)

			configPath := filepath.Join(t.TempDir(), "config.json")

			if !tt.missingFile {
				err := os.WriteFile(
					configPath,
					[]byte(tt.configJSON),
					0600,
				)
				require.NoError(t, err)
			}

			cfg := tt.config
			cfg.ConfigFile = configPath

			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			if len(tt.envVars) > 0 {
				err := env.Parse(&cfg)
				require.NoError(t, err)
			}

			err := readConfigFromFile(&cfg, tt.providedFlags)

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
