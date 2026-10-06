package notifier

import (
	"errors"
	"testing"

	"github.com/bazueva/metrics/internal/notifier/events"
	"github.com/bazueva/metrics/internal/notifier/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestNotifier_Notify(t *testing.T) {
	t.Parallel()

	t.Run("успешное добавление события в очередь", func(t *testing.T) {
		t.Parallel()

		subscriber := mocks.NewMockSubscriber(t)

		n := NewNotifier(
			[]Subscriber{subscriber},
			zap.NewNop(),
		)

		event := events.MetricsSavedEvent{
			TS:        123,
			Metrics:   []string{"metric1"},
			IPAddress: "127.0.0.1",
		}

		err := n.Notify(event)

		require.NoError(t, err)
		require.Equal(t, event, <-n.eventCh)
	})

	t.Run("нет подписчиков", func(t *testing.T) {
		t.Parallel()

		n := NewNotifier(nil, zap.NewNop())

		for i := 0; i < 10; i++ {
			err := n.Notify(events.MetricsSavedEvent{})
			require.NoError(t, err)
		}

		require.Empty(t, n.eventCh)
	})

	t.Run("очередь переполнена", func(t *testing.T) {
		t.Parallel()

		subscriber := mocks.NewMockSubscriber(t)

		n := NewNotifier(
			[]Subscriber{subscriber},
			zap.NewNop(),
		)

		for i := 0; i < workers; i++ {
			err := n.Notify(events.MetricsSavedEvent{})
			require.NoError(t, err)
		}

		err := n.Notify(events.MetricsSavedEvent{})

		require.ErrorIs(t, err, errQueueFull)
	})
}

func TestNotifier_StartStop(t *testing.T) {
	t.Parallel()

	t.Run("событие отправляется подписчику", func(t *testing.T) {
		t.Parallel()

		subscriber := mocks.NewMockSubscriber(t)

		event := events.MetricsSavedEvent{
			TS:        123,
			Metrics:   []string{"metric1"},
			IPAddress: "127.0.0.1",
		}

		subscriber.
			EXPECT().
			OnMetricsSaved(event).
			Return(nil)

		n := NewNotifier(
			[]Subscriber{subscriber},
			zap.NewNop(),
		)

		n.Start()

		err := n.Notify(event)
		require.NoError(t, err)

		n.Stop()
	})

	t.Run("событие отправляется всем подписчикам", func(t *testing.T) {
		t.Parallel()

		firstSubscriber := mocks.NewMockSubscriber(t)
		secondSubscriber := mocks.NewMockSubscriber(t)

		event := events.MetricsSavedEvent{
			TS:        123,
			Metrics:   []string{"metric1"},
			IPAddress: "127.0.0.1",
		}

		firstSubscriber.
			EXPECT().
			OnMetricsSaved(event).
			Return(nil)

		secondSubscriber.
			EXPECT().
			OnMetricsSaved(event).
			Return(nil)

		n := NewNotifier(
			[]Subscriber{
				firstSubscriber,
				secondSubscriber,
			},
			zap.NewNop(),
		)

		n.Start()

		err := n.Notify(event)
		require.NoError(t, err)

		n.Stop()
	})

	t.Run("ошибка одного подписчика не мешает остальным", func(t *testing.T) {
		t.Parallel()

		core, logs := observer.New(zapcore.ErrorLevel)
		logger := zap.New(core)

		firstSubscriber := mocks.NewMockSubscriber(t)
		secondSubscriber := mocks.NewMockSubscriber(t)

		event := events.MetricsSavedEvent{
			TS:        123,
			Metrics:   []string{"metric1"},
			IPAddress: "127.0.0.1",
		}

		subscriberErr := errors.New("subscriber error")

		firstSubscriber.
			EXPECT().
			OnMetricsSaved(event).
			Return(subscriberErr)

		secondSubscriber.
			EXPECT().
			OnMetricsSaved(event).
			Return(nil)

		n := NewNotifier(
			[]Subscriber{
				firstSubscriber,
				secondSubscriber,
			},
			logger,
		)

		n.Start()

		err := n.Notify(event)
		require.NoError(t, err)

		n.Stop()

		require.Len(t, logs.All(), 1)
		entry := logs.All()[0]

		require.Equal(t, "error send notify", entry.Message)
		require.Equal(t, subscriberErr.Error(), entry.ContextMap()["error"])
	})
}
