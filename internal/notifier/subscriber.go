package notifier

// Subscriber определяет интерфейс подписчика на события сохранения метрик.
type Subscriber interface {
	OnMetricsSaved(event MetricsSavedEvent) error
}
