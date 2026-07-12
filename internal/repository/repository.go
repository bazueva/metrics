package repository

import (
	"encoding/json"
	"fmt"
	"net/http"

	models "github.com/bazueva/metrics/internal/model"
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

func (r *repository) SendMetric(metric models.Metrics) error {
	updateUrl := fmt.Sprintf("%s/update", r.addr)

	metricJson, err := json.Marshal(metric)
	if err != nil {
		return err
	}

	response, err := r.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(metricJson).
		Post(updateUrl)
	if err != nil {
		return err
	}

	if response.StatusCode() != http.StatusOK {
		return fmt.Errorf("Ошибка отправки метрик: статус - %d, ответ - %s", response.StatusCode(), response.String())
	}

	return nil
}
