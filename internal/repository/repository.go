package repository

import (
	"fmt"

	resty "github.com/go-resty/resty/v2"
)

type MetricSender interface {
	SendMetric(metricType string, metricName string, metricValue string) error
}

type repository struct {
	addr string
}

func NewRepository(addr string) (*repository, error) {
	if addr == "" {
		return nil, fmt.Errorf("Не указан адрес сервера")
	}

	return &repository{
		addr: addr,
	}, nil
}

func (r *repository) SendMetric(metricType string, metricName string, metricValue string) error {
	updateUrl := fmt.Sprintf("%s/update/%s/%s/%s", r.addr, metricType, metricName, metricValue)

	client := resty.New()

	_, err := client.R().
		SetHeader("Content-Type", "text/plain").
		Post(updateUrl)
	if err != nil {
		return err
	}

	return nil
}
