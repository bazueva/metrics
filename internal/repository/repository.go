package repository

import (
	"fmt"
	"net/http"

	resty "github.com/go-resty/resty/v2"
)

type repository struct {
	addr   string
	client *resty.Client
}

func NewRepository(addr string) (*repository, error) {
	if addr == "" {
		return nil, fmt.Errorf("Не указан адрес сервера")
	}

	return &repository{
		addr:   addr,
		client: resty.New(),
	}, nil
}

func (r *repository) SendMetric(metricType string, metricName string, metricValue string) error {
	updateUrl := fmt.Sprintf("%s/update/%s/%s/%s", r.addr, metricType, metricName, metricValue)

	response, err := r.client.R().
		SetHeader("Content-Type", "text/plain").
		Post(updateUrl)
	if err != nil {
		return err
	}

	if response.StatusCode() != http.StatusOK {
		return fmt.Errorf("Ошибка отправки метрик: статус - %d, ответ - %s", response.StatusCode(), response.String())
	}

	return nil
}
