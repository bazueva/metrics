package notifier

import "github.com/bazueva/metrics/internal/notifier/events"

// Subscriber определяет интерфейс подписчика на события сохранения метрик.
type Subscriber interface {
	OnMetricsSaved(event events.MetricsSavedEvent) error
}
