package notifier

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

type HttpClient interface {
	Post(url string) (*resty.Response, error)
}

type HttpSubscriber struct {
	url    string
	client *resty.Client
}

func NewHTTPSubscriber(url string) *HttpSubscriber {
	return &HttpSubscriber{
		url:    url,
		client: resty.New(),
	}
}

func (s *HttpSubscriber) OnMetricsSaved(event MetricsSavedEvent) error {
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
