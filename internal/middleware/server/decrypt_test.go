package server

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bazueva/metrics/internal/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecryptRequest(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		original := []byte(`{"id":"metric1","type":"gauge","value":12.5}`)

		encrypted, err := crypto.Encrypt(&privateKey.PublicKey, original)
		require.NoError(t, err)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			assert.Equal(t, original, body)
			assert.Equal(t, int64(len(original)), r.ContentLength)

			w.WriteHeader(http.StatusOK)
		})

		request := httptest.NewRequest(
			http.MethodPost,
			"/updates/",
			bytes.NewReader(encrypted),
		)
		request.Header.Set("X-Encrypted", "true")

		recorder := httptest.NewRecorder()

		DecryptRequest(privateKey)(handler).ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("wrong private key", func(t *testing.T) {
		t.Parallel()

		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		anotherPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		original := []byte(`{"id":"metric1","type":"gauge","value":12.5}`)

		encrypted, err := crypto.Encrypt(&privateKey.PublicKey, original)
		require.NoError(t, err)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("handler should not be called")
		})

		request := httptest.NewRequest(
			http.MethodPost,
			"/updates/",
			bytes.NewReader(encrypted),
		)
		request.Header.Set("X-Encrypted", "true")

		recorder := httptest.NewRecorder()

		DecryptRequest(anotherPrivateKey)(handler).ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})

	t.Run("without private key", func(t *testing.T) {
		t.Parallel()

		original := []byte(`{"id":"metric1","type":"gauge","value":12.5}`)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			assert.Equal(t, original, body)

			w.WriteHeader(http.StatusOK)
		})

		request := httptest.NewRequest(
			http.MethodPost,
			"/updates/",
			bytes.NewReader(original),
		)
		request.Header.Set("X-Encrypted", "true")

		recorder := httptest.NewRecorder()

		DecryptRequest(nil)(handler).ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})

	t.Run("without X-Encrypted header", func(t *testing.T) {
		t.Parallel()

		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		original := []byte(`{"id":"metric1","type":"gauge","value":12.5}`)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			assert.Equal(t, original, body)

			w.WriteHeader(http.StatusOK)
		})

		request := httptest.NewRequest(
			http.MethodPost,
			"/updates/",
			bytes.NewReader(original),
		)

		recorder := httptest.NewRecorder()

		DecryptRequest(privateKey)(handler).ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusOK, recorder.Code)
	})
}
