package main

import (
	"os"
	"strings"
	"testing"
	"time"

	configpkg "github.com/bazueva/metrics/cmd/config"
	"github.com/bazueva/metrics/internal/agent"
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
				MetricServerAddr: configpkg.ServerAddr{
					Host: "localhost",
					Port: 8080,
				},
				ReportInterval: agent.ReportInterval,
				PollInterval:   agent.PollInterval,
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
			},
		},
		{
			name: "with REPORT_INTERVAL env",
			envVars: map[string]string{
				"REPORT_INTERVAL": "2s",
			},
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "localhost",
					Port: 8080,
				},
				ReportInterval: time.Second * 2,
				PollInterval:   agent.PollInterval,
			},
		},
		{
			name: "with POLL_INTERVAL env",
			envVars: map[string]string{
				"POLL_INTERVAL": "5s",
			},
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "localhost",
					Port: 8080,
				},
				ReportInterval: agent.ReportInterval,
				PollInterval:   time.Second * 5,
			},
		},
		{
			name: "with all envs",
			envVars: map[string]string{
				"POLL_INTERVAL":   "5s",
				"REPORT_INTERVAL": "5s",
				"ADDRESS":         "test:8900",
			},
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "test",
					Port: 8900,
				},
				ReportInterval: time.Second * 5,
				PollInterval:   time.Second * 5,
			},
		},
		{
			name: "with args",
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "local",
					Port: 1111,
				},
				ReportInterval: time.Second * 48,
				PollInterval:   time.Second * 19,
			},
			args: []string{"cmd", "-a", "local:1111", "-p", "19s", "-r", "48s"},
		},
		{
			name: "with args and envs",
			envVars: map[string]string{
				"POLL_INTERVAL":   "5s",
				"REPORT_INTERVAL": "5s",
				"ADDRESS":         "test:8900",
			},
			want: config{
				MetricServerAddr: configpkg.ServerAddr{
					Host: "test",
					Port: 8900,
				},
				ReportInterval: time.Second * 5,
				PollInterval:   time.Second * 5,
			},
			args: []string{"cmd", "-a", "local:1111", "-p", "19s", "-r", "48s"},
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
