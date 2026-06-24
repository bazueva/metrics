package agent

import (
	"strconv"

	models "github.com/bazueva/metrics/internal/model"
)

type SenderRepository interface {
	SendMetric(metricType string, metricName string, metricValue string) error
}

type sender struct {
	repository SenderRepository
}

func NewSender(repository SenderRepository) *sender {
	return &sender{repository: repository}
}

func (s *sender) SendSnapshot(metrics []models.Metrics) error {
	var err error
	for _, value := range metrics {
		var metricValue string
		if value.MType == models.Counter {
			metricValue = strconv.FormatInt(*value.Delta, 10)
		} else {
			metricValue = strconv.FormatFloat(*value.Value, 'f', -1, 64)
		}

		err = s.repository.SendMetric(value.MType, value.ID, metricValue)
		if err != nil {
			return err
		}
	}

	return nil
}
