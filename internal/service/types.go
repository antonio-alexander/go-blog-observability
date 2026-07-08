package service

import "net/http"

const packageName string = "github.com/antonio-alexander/go-blog-observability/internal/service"

type middleware func(http.Handler) http.Handler

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
	bytes      []byte
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriterWrapper) Write(bytes []byte) (int, error) {
	rw.bytes = bytes
	return rw.ResponseWriter.Write(bytes)
}
