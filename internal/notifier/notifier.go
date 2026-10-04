package notifier

import "errors"

// Notifier отправляет события всем зарегистрированным подписчикам.
type Notifier struct {
	subscribers []Subscriber
}

// NewNotifier создаёт новый Notifier с указанными подписчиками.
func NewNotifier(subscribers ...Subscriber) *Notifier {
	return &Notifier{
		subscribers: subscribers,
	}
}

// Notify отправляет событие всем зарегистрированным подписчикам.
// Ошибки подписчиков объединяются и возвращаются после завершения отправки.
func (n *Notifier) Notify(event MetricsSavedEvent) error {
	var err error

	for _, subscriber := range n.subscribers {
		if subscriberErr := subscriber.OnMetricsSaved(event); subscriberErr != nil {
			err = errors.Join(err, subscriberErr)
		}
	}

	return err
}
