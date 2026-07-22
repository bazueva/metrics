package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	return g.Writer.Write(b)
}

func ServerResponseGzip() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			//contentType := r.Header.Get("Content-Type")

			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				//(contentType != "application/json" && contentType != "text/html" && contentType != "") {
				next.ServeHTTP(w, r)

				return
			}

			gzipWriter := gzip.NewWriter(w)
			defer gzipWriter.Close()

			gzResponseWriter := &gzipResponseWriter{
				ResponseWriter: w,
				Writer:         gzipWriter,
			}

			gzResponseWriter.Header().Set("Content-Encoding", "gzip")

			next.ServeHTTP(gzResponseWriter, r)
		}

		return http.HandlerFunc(fn)
	}
}
