package notifier

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileSubscriber_OnMetricsSaved(t *testing.T) {
	t.Run("записывает событие в файл", func(t *testing.T) {
		filePath := filepath.Join(t.TempDir(), "audit.log")

		subscriber := NewFileSubscriber(filePath)

		event := MetricsSavedEvent{
			TS:        12345678,
			Metrics:   []string{"Alloc", "Frees"},
			IPAddress: "192.168.0.42",
		}

		err := subscriber.OnMetricsSaved(event)
		require.NoError(t, err)

		data, err := os.ReadFile(filePath)
		require.NoError(t, err)

		assert.Equal(
			t,
			`{"ts":12345678,"metrics":["Alloc","Frees"],"ip_address":"192.168.0.42"}`+"\n",
			string(data),
		)
	})

	t.Run("добавляет событие в существующий файл", func(t *testing.T) {
		filePath := filepath.Join(t.TempDir(), "audit.log")

		subscriber := NewFileSubscriber(filePath)

		firstEvent := MetricsSavedEvent{
			TS:        1,
			Metrics:   []string{"Alloc"},
			IPAddress: "127.0.0.1",
		}

		secondEvent := MetricsSavedEvent{
			TS:        2,
			Metrics:   []string{"Frees"},
			IPAddress: "127.0.0.2",
		}

		require.NoError(t, subscriber.OnMetricsSaved(firstEvent))
		require.NoError(t, subscriber.OnMetricsSaved(secondEvent))

		data, err := os.ReadFile(filePath)
		require.NoError(t, err)

		assert.Equal(
			t,
			`{"ts":1,"metrics":["Alloc"],"ip_address":"127.0.0.1"}`+"\n"+
				`{"ts":2,"metrics":["Frees"],"ip_address":"127.0.0.2"}`+"\n",
			string(data),
		)
	})

	t.Run("возвращает ошибку если путь недоступен", func(t *testing.T) {
		subscriber := NewFileSubscriber("/directory/does/not/exist/audit.log")

		err := subscriber.OnMetricsSaved(MetricsSavedEvent{
			TS:        1,
			Metrics:   []string{"Alloc"},
			IPAddress: "127.0.0.1",
		})

		require.Error(t, err)
	})
}
