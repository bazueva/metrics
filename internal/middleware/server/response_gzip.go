package server

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
)

var gzipWriterPool = sync.Pool{
	New: func() any {
		return gzip.NewWriter(io.Discard)
	},
}

type gzipResponseWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	return g.Writer.Write(b)
}

func ResponseGzip() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)

				return
			}

			gzipWriter := gzipWriterPool.Get().(*gzip.Writer)
			gzipWriter.Reset(w)
			defer func() {
				_ = gzipWriter.Close()
				gzipWriterPool.Put(gzipWriter)
			}()

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
