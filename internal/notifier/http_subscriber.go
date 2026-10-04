package notifier

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

// HTTPSubscriber отправляет события аудита на удалённый HTTP-сервер.
type HTTPSubscriber struct {
	url    string
	client *resty.Client
}

// NewHTTPSubscriber создаёт подписчика, отправляющего события аудита
// по указанному URL.
func NewHTTPSubscriber(url string) *HTTPSubscriber {
	return &HTTPSubscriber{
		url:    url,
		client: resty.New(),
	}
}

// OnMetricsSaved отправляет событие сохранения метрик на сервер аудита.
func (s *HTTPSubscriber) OnMetricsSaved(event MetricsSavedEvent) error {
	response, err := s.client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(event).
		Post(s.url)
	if err != nil {
		return fmt.Errorf("ошибка отправки сообщения аудита: %w", err)
	}

	if response.IsError() {
		return fmt.Errorf("сервер аудита вернул статус %d", response.StatusCode())
	}

	return nil
}
