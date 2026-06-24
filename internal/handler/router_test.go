package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	memStorage "github.com/bazueva/metrics/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockStorage struct {
	err error
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
			name:       "not allowed method",
			request:    httptest.NewRequest(http.MethodGet, "http://test", nil),
			memStorage: nil,
			want: want{
				code: http.StatusMethodNotAllowed,
				body: "Method Not Allowed\n",
			},
		},
		{
			name:       "metric type not specified",
			request:    httptest.NewRequest(http.MethodPost, "http://test", nil),
			memStorage: nil,
			want: want{
				code: http.StatusNotFound,
				body: "",
			},
		},
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
