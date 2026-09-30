package notifier

import "errors"

type Notifier struct {
	subscribers []Subscriber
}

func NewNotifier(subscribers ...Subscriber) *Notifier {
	return &Notifier{
		subscribers: subscribers,
	}
}

func (n *Notifier) Notify(event MetricsSavedEvent) error {
	var err error

	for _, subscriber := range n.subscribers {
		if subscriberErr := subscriber.OnMetricsSaved(event); subscriberErr != nil {
			err = errors.Join(err, subscriberErr)
		}
	}

	return err
}
