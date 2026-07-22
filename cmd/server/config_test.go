package main

import (
	"os"
	"strings"
	"testing"

	configpkg "github.com/bazueva/metrics/cmd/config"
	"github.com/stretchr/testify/assert"
)

func Test_readConfig(t *testing.T) {
	type test struct {
		name    string
		envVars map[string]string
		args    []string
		want    config
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
			want: config{
				ServerAddr: configpkg.ServerAddr{
					Host: "local",
					Port: 1111,
				},
				StoreInterval:       20,
				FileStoragePath:     "f.log",
				LoadMetricsFromFile: true,
			},
			args: []string{"cmd", "-a", "local:1111", "-i", "20", "-f", "f.log", "-r", "true"},
		},
		{
			name: "with args and env",
			envVars: map[string]string{
				"ADDRESS":           "test:8900",
				"STORE_INTERVAL":    "56",
				"FILE_STORAGE_PATH": "file_env.log",
				"RESTORE":           "true",
			},
			want: config{
				ServerAddr: configpkg.ServerAddr{
					Host: "test",
					Port: 8900,
				},
				StoreInterval:       56,
				FileStoragePath:     "file_env.log",
				LoadMetricsFromFile: true,
			},
			args: []string{"cmd", "-a", "local:1111", "-i", "20", "-f", "f.log", "-r", "false"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			os.Args = tt.args

			oldEnv := os.Environ()
			defer func() {
				os.Clearenv()
				for _, env := range oldEnv {
					parts := strings.SplitN(env, "=", 2)
					os.Setenv(parts[0], parts[1])
				}
			}()

			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			cfg, err := readConfig()

			assert.Nil(t, err)
			assert.Equal(t, tt.want, cfg)
		})
	}
}
