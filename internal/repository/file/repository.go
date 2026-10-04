package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	models "github.com/bazueva/metrics/internal/model"
)

// Repository реализует файловое хранилище метрик.
type Repository struct {
	filename string
}

// Save сохраняет метрики в файл в формате JSON.
func (r *Repository) Save(ctx context.Context, data []models.Metrics) error {
	if len(data) == 0 {
		return nil
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("ошибка json.Marshal - %w", err)
	}

	err = os.WriteFile(r.filename, dataJSON, 0666)
	if err != nil {
		return fmt.Errorf("ошибка сохранения - %w", err)
	}

	return nil
}

// Load загружает метрики из файла.
func (r *Repository) Load(ctx context.Context) ([]models.Metrics, error) {
	data, err := os.ReadFile(r.filename)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла - %w", err)
	}

	if len(data) == 0 {
		return nil, err
	}

	var result []models.Metrics
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("ошибка json.Unmarshal - %w", err)
	}

	return result, nil
}

// NewRepository создаёт новый файловый репозиторий для хранения метрик.
func NewRepository(filename string) *Repository {
	return &Repository{
		filename: filename,
	}
}
