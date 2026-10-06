package notifier

import (
	"fmt"
	"time"

	"github.com/bazueva/metrics/internal/notifier/events"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

// HTTPSubscriber отправляет события аудита на удалённый HTTP-сервер.
type HTTPSubscriber struct {
	url    string
	client *resty.Client
}

// NewHTTPSubscriber создаёт подписчика, отправляющего события аудита
// по указанному URL.
func NewHTTPSubscriber(url string, logger *zap.Logger) *HTTPSubscriber {
	return &HTTPSubscriber{
		url: url,
		client: resty.New().
			SetTimeout(1 * time.Second).
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
			SetRetryCount(3),
	}
}

// OnMetricsSaved отправляет событие сохранения метрик на сервер аудита.
func (s *HTTPSubscriber) OnMetricsSaved(event events.MetricsSavedEvent) error {
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
