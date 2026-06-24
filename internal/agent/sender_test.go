package agent

import (
	"fmt"

	"github.com/stretchr/testify/assert"

	"testing"

	models "github.com/bazueva/metrics/internal/model"
)

type SenderRepositoryMock struct {
	err error
}

func (s SenderRepositoryMock) SendMetric(metricType string, metricName string, metricValue string) error {
	return s.err
}

func TestSender_SendSnapshot(t *testing.T) {
	type args struct {
		metrics    []models.Metrics
		repository SenderRepository
	}

	type test struct {
		name string
		args args
		err  bool
	}

	tests := []test{
		{
			name: "empty metrics",
			args: args{},
			err:  false,
		},
		{
			name: "error repository",
			args: args{
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
			args: args{
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
			testSender := NewSender(tt.args.repository)
			err := testSender.SendSnapshot(tt.args.metrics)

			if tt.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
