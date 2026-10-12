package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func BenchmarkResponseGzip(b *testing.B) {
	body := strings.Repeat("metric=value\n", 100)

	handler := ResponseGzip()(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(body))
		},
	))

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept-Encoding", "gzip")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)
	}
}
