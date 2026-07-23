package file

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	models "github.com/bazueva/metrics/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_Save(t *testing.T) {
	tmpDir := t.TempDir()
	fileName := filepath.Join(tmpDir, "test_metric.log")

	type test struct {
		name      string
		data      []models.Metrics
		fileName  string
		checkFile bool
		err       string
	}

	tests := []test{
		{
			name:     "empty data",
			data:     nil,
			fileName: fileName,
		},
		{
			name: "file not exists",
			data: []models.Metrics{
				{
					ID:    "1",
					MType: models.Gauge,
					Delta: new(int64(1)),
				},
			},
			checkFile: true,
			fileName:  fileName,
		},
		{
			name: "error write",
			data: []models.Metrics{
				{
					ID:    "1",
					MType: models.Gauge,
					Value: new(float64(1)),
				},
				{
					ID:    "2",
					MType: models.Counter,
					Delta: new(int64(10)),
				},
			},
			checkFile: false,
			err:       "Ошибка сохранения - open /test/1/rt.log: no such file or directory",
			fileName:  "/test/1/rt.log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewRepository(tt.fileName)

			err := repo.Save(nil, tt.data)
			if err != nil || tt.err != "" {
				assert.Equal(t, tt.err, err.Error())
			}

			if tt.checkFile {
				_, err = os.Stat(tt.fileName)
				assert.Nil(t, err)

				fileData, err := os.ReadFile(tt.fileName)
				require.NoError(t, err)

				var data []models.Metrics
				err = json.Unmarshal(fileData, &data)
				require.NoError(t, err)

				assert.Equal(t, data, tt.data)

			}
		})
	}
}

func TestRepository_LoadFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	fileName := filepath.Join(tmpDir, "test_metric_load.log")

	type test struct {
		name           string
		data           []models.Metrics
		needCreateFile bool
		fileName       string
		err            string
	}

	tests := []test{
		{
			name:     "file not found",
			data:     nil,
			fileName: "test/1.log",
			err:      "Ошибка чтения файла - open test/1.log: no such file or directory",
		},
		{
			name:           "empty file",
			data:           nil,
			fileName:       fileName,
			needCreateFile: true,
			err:            "",
		},
		{
			name: "success",
			data: []models.Metrics{
				{
					ID:    "1",
					MType: models.Gauge,
					Value: new(float64(1)),
				},
				{
					ID:    "2",
					MType: models.Counter,
					Delta: new(int64(10)),
				},
			},
			fileName:       fileName,
			needCreateFile: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewRepository(tt.fileName)

			if tt.needCreateFile {
				jsonData, err := json.Marshal(tt.data)
				err = os.WriteFile(tt.fileName, jsonData, 0666)
				assert.Nil(t, err)
			}

			data, err := repo.Load(nil)
			if err != nil || tt.err != "" {
				assert.Equal(t, tt.err, err.Error())

				return
			}

			assert.Equal(t, tt.data, data)
		})
	}
}
