package handler

import (
	"bytes"
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
	err              error
	metric           models.Metrics
	metrics          []models.Metrics
	createdMetric    models.Metrics
	createdMetricErr error
}

func (m *MockStorage) UpdateMetric(metric models.Metrics) error {
	return m.err
}

func (m *MockStorage) CreateMetric(metricType string, name string, value string) (models.Metrics, error) {
	return m.createdMetric, m.createdMetricErr
}

func (m *MockStorage) GetMetric(name string) (models.Metrics, error) {
	return m.metric, m.err
}

func (m *MockStorage) GetAllMetrics() []models.Metrics {
	return m.metrics
}

func TestHandler_UpdateHandler(t *testing.T) {
	type want struct {
		code int
		body string
	}

	type test struct {
		name       string
		request    *http.Request
		memStorage Storage
		want       want
	}

	tests := []test{
		{
			name:    "error create metric",
			request: httptest.NewRequest(http.MethodPost, "http://test/metricType/", nil),
			memStorage: func() Storage {
				mock := new(MockStorage)
				mock.createdMetricErr = fmt.Errorf("Ошибка создания метрики")

				return mock
			}(),
			want: want{
				code: http.StatusBadRequest,
				body: "Ошибка создания метрики\n",
			},
		},
		{
			name:    "error storage",
			request: httptest.NewRequest(http.MethodPost, "http://test/metricType/", nil),
			memStorage: func() Storage {
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
			memStorage: func() Storage {
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
			memStorage: func() Storage {
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
		memStorage Storage
		want       want
	}

	tests := []test{
		{
			name:    "error storage",
			request: httptest.NewRequest(http.MethodPost, "http://test/metricType/", nil),
			memStorage: func() Storage {
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
			memStorage: func() Storage {
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
			memStorage: func() Storage {
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
		memStorage Storage
		want       want
	}

	tests := []test{
		{
			name:    "success",
			request: httptest.NewRequest(http.MethodPost, "http://test/metricType/", nil),
			memStorage: func() Storage {
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

func TestHandler_UpdateMetricHandler(t *testing.T) {
	type want struct {
		code int
		body string
	}

	type test struct {
		name       string
		request    *http.Request
		memStorage Storage
		want       want
	}

	tests := []test{
		{
			name:       "invalid json",
			request:    httptest.NewRequest(http.MethodPost, "http://test/metricType/", bytes.NewReader([]byte(`"test"`))),
			memStorage: nil,
			want: want{
				code: http.StatusBadRequest,
				body: "json: cannot unmarshal string into Go value of type models.Metrics\n",
			},
		},
		{
			name:    "error storage update metric",
			request: httptest.NewRequest(http.MethodPost, "http://test/metricType/", bytes.NewReader([]byte(`{"test":"1"}`))),
			memStorage: func() Storage {
				mock := new(MockStorage)
				mock.err = fmt.Errorf("ошибка")

				return mock
			}(),
			want: want{
				code: http.StatusBadRequest,
				body: "ошибка\n",
			},
		},
		{
			name:    "success",
			request: httptest.NewRequest(http.MethodPost, "http://test/metricType/", bytes.NewReader([]byte(`{"test":"1"}`))),
			memStorage: func() Storage {
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

			handler.UpdateMetricHandler(recorder, tt.request)

			result := recorder.Result()
			defer result.Body.Close()

			body, err := io.ReadAll(result.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.want.code, result.StatusCode)
			assert.Equal(t, tt.want.body, string(body))
		})
	}
}

func TestHandler_ValueMetricHandler(t *testing.T) {
	type want struct {
		code int
		body string
	}

	type test struct {
		name       string
		request    *http.Request
		memStorage Storage
		want       want
	}

	tests := []test{
		{
			name:       "invalid json",
			request:    httptest.NewRequest(http.MethodPost, "http://test/metricType/", bytes.NewReader([]byte(`"test"`))),
			memStorage: nil,
			want: want{
				code: http.StatusBadRequest,
				body: `{"error":"json: cannot unmarshal string into Go value of type models.Metrics"}`,
			},
		},
		{
			name:    "error storage get metric",
			request: httptest.NewRequest(http.MethodPost, "http://test/metricType/", bytes.NewReader([]byte(`{"test":"1"}`))),
			memStorage: func() Storage {
				mock := new(MockStorage)
				mock.err = memStorage.ErrNotFoundMetric

				return mock
			}(),
			want: want{
				code: http.StatusNotFound,
				body: `{"error":"not found"}`,
			},
		},
		{
			name:    "success",
			request: httptest.NewRequest(http.MethodPost, "http://test/metricType/", bytes.NewReader([]byte(`{"test":"1"}`))),
			memStorage: func() Storage {
				mock := new(MockStorage)
				mock.metric = models.Metrics{
					ID:    "test",
					MType: models.Gauge,
					Delta: nil,
					Value: new(float64(1)),
				}

				return mock
			}(),
			want: want{
				code: http.StatusOK,
				body: `{"id":"test","type":"gauge","value":1}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(tt.memStorage)
			recorder := httptest.NewRecorder()

			handler.ValueMetricHandler(recorder, tt.request)

			result := recorder.Result()
			defer result.Body.Close()

			body, err := io.ReadAll(result.Body)
			require.NoError(t, err)

			assert.Equal(t, tt.want.code, result.StatusCode)
			assert.JSONEq(t, tt.want.body, string(body))
			assert.Equal(t, "application/json", result.Header.Get("Content-Type"))
		})
	}
}
