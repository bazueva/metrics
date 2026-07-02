package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	models "github.com/bazueva/metrics/internal/model"
	memStorage "github.com/bazueva/metrics/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockStorage struct {
	err     error
	metric  models.Metrics
	metrics []models.Metrics
}

func (m *MockStorage) GetMetric(name string) (models.Metrics, error) {
	return m.metric, m.err
}

func (m *MockStorage) GetAllMetrics() []models.Metrics {
	return m.metrics
}

func (m *MockStorage) UpdateMetric(metricType string, name string, value string) error {
	return m.err
}

func TestHandler_UpdateHandler(t *testing.T) {
	type want struct {
		code int
		body string
	}

	type test struct {
		name       string
		request    *http.Request
		memStorage memStorage.Storage
		want       want
	}

	tests := []test{
		{
			name:    "error storage",
			request: httptest.NewRequest(http.MethodPost, "http://test/metricType/", nil),
			memStorage: func() memStorage.Storage {
				mock := new(MockStorage)
				mock.err = fmt.Errorf("Ошибка")

				return mock
			}(),
			want: want{
				code: http.StatusBadRequest,
				body: "Ошибка\n",
			},
		},
		{
			name:    "error empty metric from storage",
			request: httptest.NewRequest(http.MethodPost, "http://test/metricType/", nil),
			memStorage: func() memStorage.Storage {
				mock := new(MockStorage)
				mock.err = memStorage.ErrEmptyMetricName

				return mock
			}(),
			want: want{
				code: http.StatusNotFound,
				body: "empty metric name\n",
			},
		},
		{
			name:    "success",
			request: httptest.NewRequest(http.MethodPost, "http://test/metricType/", nil),
			memStorage: func() memStorage.Storage {
				mock := new(MockStorage)

				return mock
			}(),
			want: want{
				code: http.StatusOK,
				body: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(tt.memStorage)
			recorder := httptest.NewRecorder()

			handler.UpdateHandler(recorder, tt.request)

			result := recorder.Result()
			defer result.Body.Close()

			body, err := io.ReadAll(result.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.want.code, result.StatusCode)
			assert.Equal(t, tt.want.body, string(body))
		})
	}
}

func TestHandler_GetMetricHandler(t *testing.T) {
	type want struct {
		code int
		body string
	}

	type test struct {
		name       string
		request    *http.Request
		memStorage memStorage.Storage
		want       want
	}

	tests := []test{
		{
			name:    "error storage",
			request: httptest.NewRequest(http.MethodPost, "http://test/metricType/", nil),
			memStorage: func() memStorage.Storage {
				mock := new(MockStorage)
				mock.err = fmt.Errorf("Ошибка")

				return mock
			}(),
			want: want{
				code: http.StatusBadRequest,
				body: "Ошибка\n",
			},
		},
		{
			name:    "error empty metric from storage",
			request: httptest.NewRequest(http.MethodPost, "http://test/metricType/", nil),
			memStorage: func() memStorage.Storage {
				mock := new(MockStorage)
				mock.err = memStorage.ErrEmptyMetricName

				return mock
			}(),
			want: want{
				code: http.StatusNotFound,
				body: "empty metric name\n",
			},
		},
		{
			name:    "success",
			request: httptest.NewRequest(http.MethodPost, "http://test/metricType/", nil),
			memStorage: func() memStorage.Storage {
				mock := new(MockStorage)
				mock.metric = models.Metrics{
					ID:    "test",
					MType: models.Counter,
					Delta: new(int64(1000)),
				}

				return mock
			}(),
			want: want{
				code: http.StatusOK,
				body: "1000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(tt.memStorage)
			recorder := httptest.NewRecorder()

			handler.GetMetricHandler(recorder, tt.request)

			result := recorder.Result()
			defer result.Body.Close()

			body, err := io.ReadAll(result.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.want.code, result.StatusCode)
			assert.Equal(t, tt.want.body, string(body))
		})
	}
}

func TestHandler_GetAllMetricsHandler(t *testing.T) {
	type want struct {
		code int
		body string
	}

	type test struct {
		name       string
		request    *http.Request
		memStorage memStorage.Storage
		want       want
	}

	tests := []test{
		{
			name:    "success",
			request: httptest.NewRequest(http.MethodPost, "http://test/metricType/", nil),
			memStorage: func() memStorage.Storage {
				mock := new(MockStorage)
				mock.metrics = []models.Metrics{
					{
						ID:    "test",
						MType: models.Counter,
						Delta: new(int64(1000)),
					},
					{
						ID:    "test 2",
						MType: models.Gauge,
						Value: new(1.22),
					},
					{
						ID:    "test 3",
						MType: models.Gauge,
						Value: new(9.622),
					},
				}

				return mock
			}(),
			want: want{
				code: http.StatusOK,
				body: "test - 1000 \n" +
					"test 2 - 1.220000 \n" +
					"test 3 - 9.622000 \n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(tt.memStorage)
			recorder := httptest.NewRecorder()

			handler.GetAllMetricsHandler(recorder, tt.request)

			result := recorder.Result()
			defer result.Body.Close()

			body, err := io.ReadAll(result.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.want.code, result.StatusCode)
			assert.Equal(t, tt.want.body, string(body))
		})
	}
}
