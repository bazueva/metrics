// Package metric реализует HTTP-репозиторий для отправки метрик на сервер.
package metric

import (
	"bytes"
	"compress/gzip"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bazueva/metrics/internal/crypto"
	"github.com/bazueva/metrics/internal/helpers"
	"github.com/bazueva/metrics/internal/interfaces"
	models "github.com/bazueva/metrics/internal/model"
	resty "github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

// generate:reset
type repository struct {
	addr      string
	client    *resty.Client
	secretKey string
	publicKey *rsa.PublicKey
}

// NewRepository создаёт новый репозиторий для отправки метрик на сервер.
func NewRepository(
	addr string,
	secretKey string,
	logger interfaces.Logger,
	publicKey *rsa.PublicKey,
) (*repository, error) {
	if addr == "" {
		return nil, fmt.Errorf("не указан адрес сервера")
	}

	return &repository{
		addr:      addr,
		client:    createClient(logger),
		secretKey: secretKey,
		publicKey: publicKey,
	}, nil
}

func createClient(logger interfaces.Logger) *resty.Client {
	return resty.New().
		SetTimeout(10 * time.Second).
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

// SendBatchMetric отправляет пакет метрик на сервер в формате JSON с использованием gzip-сжатия.
func (r *repository) SendBatchMetric(metrics []models.Metrics) error {
	updateURL := fmt.Sprintf("%s/updates/", r.addr)

	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return err
	}

	request := r.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Content-Encoding", "gzip")
	if r.publicKey != nil {
		request.SetHeader("X-Encrypted", "true")
	}

	r.signData(metricsJSON, request)

	compress, err := compressData(metricsJSON)
	if err != nil {
		return err
	}

	body := compress

	if r.publicKey != nil {
		body, err = crypto.Encrypt(r.publicKey, compress)
		if err != nil {
			return fmt.Errorf("encrypt metrics: %w", err)
		}
	}

	response, err := request.
		SetBody(body).
		Post(updateURL)
	if err != nil {
		return err
	}

	if response.StatusCode() != http.StatusOK {
		return fmt.Errorf("ошибка отправки метрик: статус - %d, ответ - %s", response.StatusCode(), response.String())
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
