package notifier

type Subscriber interface {
	OnMetricsSaved(event MetricsSavedEvent) error
}
