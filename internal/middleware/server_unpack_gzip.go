package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type Logger interface {
	Error(msg string, fields ...zap.Field)
}

func ServerUnpackGzip(logger Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
				next.ServeHTTP(w, r)

				return
			}

			reader, err := gzip.NewReader(r.Body)
			if err != nil {
				logger.Error("error server compress middleware", zap.Error(err))

				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("error server compress middleware"))

				return
			}
			defer r.Body.Close()
			defer reader.Close()

			body, err := io.ReadAll(reader)
			if err != nil {
				logger.Error("error server compress middleware", zap.Error(err))

				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("error server compress middleware"))

				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			r.ContentLength = int64(len(body))

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
