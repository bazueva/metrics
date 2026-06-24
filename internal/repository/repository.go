package repository

import (
	"fmt"
	"net/http"
)

const url = "http://localhost:8080"

type MetricSender interface {
	SendMetric(metricType string, metricName string, metricValue string) error
}

type repository struct {
}

func NewRepository() *repository {
	return &repository{}
}

func (r *repository) SendMetric(metricType string, metricName string, metricValue string) error {
	updateUrl := fmt.Sprintf("%s/update/%s/%s/%s", url, metricType, metricName, metricValue)

	response, err := http.Post(updateUrl, "text/plain", nil)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	return nil
}
