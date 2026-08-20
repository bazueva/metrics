package metric

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bazueva/metrics/internal/helpers"
	"github.com/bazueva/metrics/internal/interfaces"
	models "github.com/bazueva/metrics/internal/model"
	resty "github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type repository struct {
	addr      string
	client    *resty.Client
	secretKey string
}

func NewRepository(addr string, secretKey string, logger interfaces.Logger) (*repository, error) {
	if addr == "" {
		return nil, fmt.Errorf("Не указан адрес сервера")
	}

	return &repository{
		addr:      addr,
		client:    createClient(logger),
		secretKey: secretKey,
	}, nil
}

func createClient(logger interfaces.Logger) *resty.Client {
	return resty.New().
		SetRetryCount(3).
		SetRetryAfter(func(client *resty.Client, response *resty.Response) (time.Duration, error) {
			attempt := response.Request.Attempt
			delay := time.Duration(2*attempt-1) * time.Second

			logger.Info("Попытка повторного запроса",
				zap.Int("attempt", attempt),
				zap.Duration("delay", delay),
				zap.String("time", time.Now().Format("15:04:05")),
			)

			return delay, nil
		}).
		SetRetryMaxWaitTime(5 * time.Second).
		AddRetryHook(
			func(r *resty.Response, err error) {
				logger.Info("Повторная попытка...",
					zap.Error(err),
					zap.String("url", r.Request.URL),
				)
			},
		)
}

func (r *repository) SendBatchMetric(metrics []models.Metrics) error {
	updateUrl := fmt.Sprintf("%s/updates/", r.addr)

	metricsJson, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	request := r.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip")

	r.signData(metricsJson, request)

	compress, err := compressData(metricsJson)
	if err != nil {
		return err
	}

	response, err := request.
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

func (r *repository) signData(data []byte, request *resty.Request) {
	if r.secretKey == "" {
		return
	}

	request.SetHeader("HashSHA256", helpers.GenerateHMAC(r.secretKey, data))
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
