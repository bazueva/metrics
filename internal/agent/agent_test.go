package agent

import (
	"fmt"
	"testing"
	"time"

	models "github.com/bazueva/metrics/internal/model"
	"github.com/stretchr/testify/assert"
)

type SenderRepositoryMock struct {
	err       error
	callCount int
}

func (s *SenderRepositoryMock) SendMetric(metricType string, metricName string, metricValue string) error {
	s.callCount++
	return s.err
}

type MetricsSnapshotMock struct {
	metrics   []models.Metrics
	callCount int
}

func (m *MetricsSnapshotMock) MetricsSnapshot(counter int64) []models.Metrics {
	m.callCount++
	return m.metrics
}

func TestSender_sendSnapshot(t *testing.T) {
	type test struct {
		name  string
		agent agent
		err   bool
	}

	tests := []test{
		{
			name:  "empty metrics",
			agent: agent{},
			err:   false,
		},
		{
			name: "error repository",
			agent: agent{
				metrics: []models.Metrics{
					{
						ID:    "test",
						MType: models.Counter,
						Delta: new(int64(1)),
					},
				},
				repository: func() SenderRepository {
					mock := &SenderRepositoryMock{err: fmt.Errorf("ошибка")}

					return mock
				}(),
			},
			err: true,
		},
		{
			name: "success",
			agent: agent{
				metrics: []models.Metrics{
					{
						ID:    "test",
						MType: models.Counter,
						Delta: new(int64(1)),
					},
				},
				repository: func() SenderRepository {
					return new(SenderRepositoryMock)
				}(),
			},
			err: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.agent.sendSnapshot()

			if tt.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_updateMetric(t *testing.T) {
	type test struct {
		name  string
		agent agent
		want  []models.Metrics
	}

	tests := []test{
		{
			name: "success",
			agent: agent{
				collector: &MetricsSnapshotMock{metrics: []models.Metrics{
					{
						ID: "test",
					},
				}},
			},
			want: []models.Metrics{
				{
					ID: "test",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.agent.updateMetric(0)
			assert.Equal(t, tt.want, tt.agent.metrics)
		})
	}
}

func TestAgent_Run(t *testing.T) {
	a := NewAgent(
		&MetricsSnapshotMock{metrics: []models.Metrics{
			{
				ID:    "test",
				MType: models.Gauge,
				Value: new(float64(1)),
			},
		}},
		&SenderRepositoryMock{},
		1,
		2,
	)

	go a.Run()

	time.Sleep(3 * time.Second)

	assert.Greater(t, a.collector.(*MetricsSnapshotMock).callCount, 0)
	assert.Greater(t, a.repository.(*SenderRepositoryMock).callCount, 0)
}
