package repository

import (
	"fmt"

	resty "github.com/go-resty/resty/v2"
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

	client := resty.New()

	_, err := client.R().
		SetHeader("Content-Type", "text/plain").
		Post(updateUrl)
	if err != nil {
		return err
	}

	return nil
}
