package agent

import (
	"errors"
	"testing"
	"time"

	"github.com/bazueva/metrics/internal/agent/mocks"
	models "github.com/bazueva/metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type SenderRepositoryMock struct {
	err       error
	callCount int
}

func (s *SenderRepositoryMock) SendBatchMetric(metrics []models.Metrics) error {
	s.callCount++
	return s.err
}

func (s *SenderRepositoryMock) SendMetric(metric models.Metrics) error {
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
	t.Run("empty metrics", func(t *testing.T) {
		testAgent := agent{}

		err := testAgent.sendSnapshot()
		assert.Nil(t, err)
	})

	t.Run("repository error", func(t *testing.T) {
		mockRepo := mocks.NewMockSenderRepository(t)

		mockRepo.EXPECT().
			SendBatchMetric(mock.Anything).
			Return(errors.New("repository error")).
			Times(1)

		testAgent := &agent{
			metrics: []models.Metrics{
				{
					ID:    "test",
					MType: models.Counter,
					Delta: new(int64(1)),
				},
			},
			repository: mockRepo,
		}

		err := testAgent.sendSnapshot()
		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		mockRepo := mocks.NewMockSenderRepository(t)

		mockRepo.EXPECT().
			SendBatchMetric(mock.Anything).
			Return(nil).
			Times(1)

		testAgent := &agent{
			metrics: []models.Metrics{
				{
					ID:    "test",
					MType: models.Counter,
					Delta: new(int64(1)),
				},
				{
					ID:    "test2",
					MType: models.Gauge,
					Value: new(float64(123.45)),
				},
			},
			repository: mockRepo,
		}

		err := testAgent.sendSnapshot()
		assert.NoError(t, err)
	})
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
	mockCollector := mocks.NewMockCollector(t)
	mockCollector.EXPECT().
		MetricsSnapshot(mock.Anything).
		Return([]models.Metrics{
			{
				ID:    "test",
				MType: models.Gauge,
				Value: new(float64(1)),
			},
		})

	mockRepository := mocks.NewMockSenderRepository(t)
	mockRepository.EXPECT().
		SendBatchMetric(mock.Anything).
		Return(nil)

	a := NewAgent(
		mockCollector,
		mockRepository,
		1,
		2,
	)

	go a.Run()

	time.Sleep(3 * time.Second)
}
