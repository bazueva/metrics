package notifier

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

type HTTPClient interface {
	Post(url string) (*resty.Response, error)
}

type HTTPSubscriber struct {
	url    string
	client *resty.Client
}

func NewHTTPSubscriber(url string) *HTTPSubscriber {
	return &HTTPSubscriber{
		url:    url,
		client: resty.New(),
	}
}

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
