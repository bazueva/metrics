package metric

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

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
		client: createClient(),
	}, nil
}

func createClient() *resty.Client {
	return resty.New().
		SetRetryCount(3).
		SetRetryAfter(func(client *resty.Client, response *resty.Response) (time.Duration, error) {
			attempt := response.Request.Attempt
			delay := time.Duration(2*attempt-1) * time.Second

			log.Printf("Попытка #%d: начинаем ожидание %v (время: %v)",
				attempt, delay, time.Now().Format("15:04:05"))

			return delay, nil
		}).
		SetRetryMaxWaitTime(5 * time.Second).
		AddRetryHook(
			func(r *resty.Response, err error) {
				log.Printf(
					"Повторная попытка... (Ошибка: %v, Адрес: %s)\n",
					err,
					r.Request.URL,
				)
			},
		)
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

func (r *repository) SendBatchMetric(metrics []models.Metrics) error {
	updateUrl := fmt.Sprintf("%s/updates/", r.addr)

	metricsJson, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	compress, err := compressData(metricsJson)
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
