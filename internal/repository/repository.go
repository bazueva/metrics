package repository

import (
	"bytes"
	"compress/gzip"
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

	compress, err := compressData(metricJson)
	if err != nil {
		return err
	}

	response, err := r.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip").
		SetBody(compress).
		Post(updateUrl)
	if err != nil {
		return err
	}

	if response.StatusCode() != http.StatusOK {
		return fmt.Errorf("Ошибка отправки метрик: статус - %d, ответ - %s", response.StatusCode(), response.String())
	}

	return nil
}

func compressData(data []byte) ([]byte, error) {
	var b bytes.Buffer

	writer := gzip.NewWriter(&b)

	_, err := writer.Write(data)
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}
