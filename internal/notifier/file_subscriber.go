package notifier

import (
	"encoding/json"
	"fmt"
	"os"
)

// FileSubscriber сохраняет события аудита о сохранении метрик в файл.
type FileSubscriber struct {
	filePath string
}

// NewFileSubscriber создаёт подписчика, записывающего события аудита
// в файл по указанному пути.
func NewFileSubscriber(filePath string) *FileSubscriber {
	return &FileSubscriber{
		filePath: filePath,
	}
}

// OnMetricsSaved записывает событие сохранения метрик в файл аудита.
func (s *FileSubscriber) OnMetricsSaved(event MetricsSavedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	file, err := os.OpenFile(
		s.filePath,
		os.O_WRONLY|os.O_CREATE|os.O_APPEND,
		0644,
	)
	if err != nil {
		return fmt.Errorf("ошибка открытия файла аудита: %w", err)
	}
	defer file.Close()

	if _, err = file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("ошибка записи файла аудита: %w", err)
	}

	return nil
}
