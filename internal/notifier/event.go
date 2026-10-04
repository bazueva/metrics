package notifier

// MetricsSavedEvent содержит данные о событии сохранения метрик.
type MetricsSavedEvent struct {
	TS        int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}
