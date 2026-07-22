package middleware

import (
	gzip2 "compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerResponseGzip(t *testing.T) {
	type test struct {
		name           string
		body           []byte
		contentType    string
		compress       bool
		wantBody       string
		acceptEncoding string
	}

	tests := []test{
		{
			name:     "without gzip",
			body:     []byte(`{"test": 1}`),
			compress: false,
			wantBody: `{"test": 1}`,
		},
		{
			name:           "with gzip",
			compress:       true,
			body:           []byte(`{"test": 1}`),
			wantBody:       `{"test": 1}`,
			acceptEncoding: "gzip",
		},
		{
			name:           "with gzip and json ",
			compress:       true,
			body:           []byte(`{"test": 1}`),
			wantBody:       `{"test": 1}`,
			contentType:    "application/json",
			acceptEncoding: "gzip",
		},
		//{
		//	name:           "with gzip and text ",
		//	compress:       false,
		//	body:           []byte(`{"test": 1}`),
		//	wantBody:       `{"test": 1}`,
		//	contentType:    "text/plain",
		//	acceptEncoding: "gzip",
		//},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(
				"POST",
				"http://localhost:8080",
				nil,
			)

			if tt.acceptEncoding != "" {
				request.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}

			if tt.contentType != "" {
				request.Header.Set("Content-Type", tt.contentType)
			}

			middleware := ServerResponseGzip()

			responseWriter := httptest.NewRecorder()

			handler := middleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Write(tt.body)
			}))

			handler.ServeHTTP(responseWriter, request)

			contentEncoding := responseWriter.Header().Get("Content-Encoding")

			if tt.compress && contentEncoding != "gzip" {
				t.Errorf("Ожидался gzip, получен %s", contentEncoding)
			}

			if !tt.compress && contentEncoding == "gzip" {
				t.Error("Не должно быть gzip, но он есть")
			}

			var body []byte
			var err error

			if tt.compress {
				gzip, err := gzip2.NewReader(responseWriter.Body)
				if err != nil {
					t.Errorf("Ошибка создания reader - %s", err.Error())
				}

				body, err = io.ReadAll(gzip)
				if err != nil {
					t.Errorf("Ошибка чтения - %s", err.Error())
				}
			} else {
				body, err = io.ReadAll(responseWriter.Body)
				if err != nil {
					t.Errorf("Ошибка чтения - %s", err.Error())
				}
			}

			assert.Equal(t, tt.wantBody, string(body))
		})
	}
}
