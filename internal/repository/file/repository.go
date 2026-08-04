package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	models "github.com/bazueva/metrics/internal/model"
)

type Repository struct {
	filename string
}

func (r *Repository) Save(ctx context.Context, data []models.Metrics) error {
	if len(data) == 0 {
		return nil
	}

	dataJson, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("Ошибка json.Marshal - %w", err)
	}

	err = os.WriteFile(r.filename, dataJson, 0666)
	if err != nil {
		return fmt.Errorf("Ошибка сохранения - %w", err)
	}

	return nil
}

func (r *Repository) Load(ctx context.Context) ([]models.Metrics, error) {
	data, err := os.ReadFile(r.filename)
	if err != nil {
		return nil, fmt.Errorf("Ошибка чтения файла - %w", err)
	}

	if len(data) == 0 {
		return nil, err
	}

	var result []models.Metrics
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("Ошибка json.Unmarshal - %w", err)
	}

	return result, nil
}

func NewRepository(filename string) *Repository {
	return &Repository{
		filename: filename,
	}
}
