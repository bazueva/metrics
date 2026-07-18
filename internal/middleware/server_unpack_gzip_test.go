package middleware

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type LoggerMock struct {
}

func (l *LoggerMock) Info(msg string, fields ...zap.Field) {
	fmt.Println(msg)
}

func TestServerGzip(t *testing.T) {
	type test struct {
		name     string
		body     []byte
		compress bool
		wantBody string
	}

	tests := []test{
		{
			name:     "without gzip",
			body:     []byte(`{"test": 1}`),
			compress: false,
			wantBody: `{"test": 1}`,
		},
		{
			name:     "with gzip",
			compress: true,
			body:     []byte(`{"test": 1}`),
			wantBody: `{"test": 1}`,
		},
		{
			name:     "empty body",
			compress: true,
			body:     nil,
			wantBody: ``,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyReader io.Reader

			if tt.compress {
				var b bytes.Buffer

				gz := gzip.NewWriter(&b)
				gz.Write(tt.body)
				gz.Close()

				bodyReader = &b
			} else {
				bodyReader = bytes.NewReader(tt.body)
			}

			request := httptest.NewRequest(
				"POST",
				"http://localhost:8080",
				bodyReader,
			)

			if tt.compress {
				request.Header.Set("Content-Encoding", "gzip")
			}

			middleware := ServerUnpackGzip(new(LoggerMock))

			responseWriter := httptest.NewRecorder()

			handler := middleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				body, err := io.ReadAll(request.Body)
				if err != nil {
					t.Errorf("ошибка чтения: %v", err)
				}
				defer request.Body.Close()

				assert.Equal(t, tt.wantBody, string(body))
				assert.Equal(t, int64(len(tt.wantBody)), request.ContentLength)
			}))

			handler.ServeHTTP(responseWriter, request)
		})
	}
}
