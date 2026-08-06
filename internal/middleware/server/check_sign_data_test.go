package server

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bazueva/metrics/internal/interfaces/mocks"
	"github.com/stretchr/testify/assert"
)

func TestCheckSignData(t *testing.T) {
	type test struct {
		name            string
		body            []byte
		wantBody        string
		secretKeyServer string
		secretKeyAgent  string
		wantStatus      int
	}

	tests := []test{
		{
			name:       "empty server secret key",
			body:       []byte(`{"test": 1}`),
			wantBody:   `{"test": 1}`,
			wantStatus: http.StatusOK,
		},
		{
			name:            "different secret keys",
			body:            []byte(`{"test": 1}`),
			secretKeyServer: "12",
			secretKeyAgent:  "56",
			wantBody:        `wrong sign data`,
			wantStatus:      http.StatusBadRequest,
		},
		{
			name:            "sign equal",
			body:            []byte(`{"test": 1}`),
			secretKeyServer: "12",
			secretKeyAgent:  "12",
			wantBody:        `{"test": 1}`,
			wantStatus:      http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(
				"POST",
				"http://localhost:8080",
				bytes.NewReader(tt.body),
			)

			if tt.secretKeyAgent != "" {
				h := hmac.New(sha256.New, []byte(tt.secretKeyAgent))
				h.Write(tt.body)

				hash := hex.EncodeToString(h.Sum(nil))
				request.Header.Set("Hashsha256", hash)
			}

			middleware := CheckSignData(tt.secretKeyServer, mocks.NewMockLogger(t))

			responseWriter := httptest.NewRecorder()

			handler := middleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.Write(tt.body)
			}))

			handler.ServeHTTP(responseWriter, request)

			body, err := io.ReadAll(responseWriter.Body)
			assert.Nil(t, err)

			assert.Equal(t, tt.wantBody, string(body))
			assert.Equal(t, tt.wantStatus, responseWriter.Code)
		})
	}
}
