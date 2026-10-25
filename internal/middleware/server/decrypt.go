package server

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"github.com/bazueva/metrics/internal/crypto"
)

func DecryptRequest(privateKey *rsa.PrivateKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if privateKey == nil || r.Header.Get("X-Encrypted") != "true" {
				next.ServeHTTP(w, r)
				return
			}

			encrypted, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read request body", http.StatusBadRequest)
				return
			}

			decrypted, err := crypto.Decrypt(privateKey, encrypted)
			if err != nil {
				http.Error(w, "failed to decrypt request body", http.StatusBadRequest)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(decrypted))
			r.ContentLength = int64(len(decrypted))

			next.ServeHTTP(w, r)
		})
	}
}
