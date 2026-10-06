package notifier

import (
	"errors"
	"sync"

	"github.com/bazueva/metrics/internal/notifier/events"
	"go.uber.org/zap"
)

var errQueueFull = errors.New("audit queue is full")

const workers = 3

// Notifier отправляет события всем зарегистрированным подписчикам.
type Notifier struct {
	subscribers []Subscriber
	eventCh     chan events.MetricsSavedEvent
	logger      *zap.Logger
	wg          sync.WaitGroup
}

func (n *Notifier) Start() {
	if len(n.subscribers) == 0 {
		return
	}

	for i := 0; i < workers; i++ {
		n.wg.Go(func() {
			n.worker()
		})
	}
}

func (n *Notifier) Stop() {
	close(n.eventCh)
	n.wg.Wait()
	n.logger.Info("Воркеры аудита завершены")
}

func (n *Notifier) worker() {
	for event := range n.eventCh {
		err := n.sendNotify(event)
		if err != nil {
			n.logger.Error("error send notify", zap.Error(err))
		}
	}
}

// NewNotifier создаёт новый Notifier с указанными подписчиками.
func NewNotifier(subscribers []Subscriber, logger *zap.Logger) *Notifier {
	return &Notifier{
		subscribers: subscribers,
		eventCh:     make(chan events.MetricsSavedEvent, workers),
		logger:      logger,
	}
}

// Notify отправляет событие в канал.
func (n *Notifier) Notify(event events.MetricsSavedEvent) error {
	if len(n.subscribers) == 0 {
		return nil
	}

	select {
	case n.eventCh <- event:
		return nil
	default:
		return errQueueFull
	}
}

func (n *Notifier) sendNotify(event events.MetricsSavedEvent) error {
	var err error

	for _, subscriber := range n.subscribers {
		if subscriberErr := subscriber.OnMetricsSaved(event); subscriberErr != nil {
			err = errors.Join(err, subscriberErr)
		}
	}

	return err
}
