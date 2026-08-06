package server

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/bazueva/metrics/internal/interfaces"
	"go.uber.org/zap"
)

func CheckSignData(secretKey string, logger interfaces.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if secretKey == "" || r.Method != http.MethodPost {
				next.ServeHTTP(w, r)

				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				logger.Error("read body error", zap.Error(err))

				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("read body error"))

				return
			}
			defer r.Body.Close()

			h := hmac.New(sha256.New, []byte(secretKey))
			h.Write(body)

			hash := hex.EncodeToString(h.Sum(nil))
			if hash != r.Header.Get("Hashsha256") {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("wrong sign data"))

				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
