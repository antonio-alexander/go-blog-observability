package service

import "net/http"

const packageName string = "github.com/antonio-alexander/go-blog-observability/internal/service"

type middleware func(http.Handler) http.Handler

type statusCodeResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (s *statusCodeResponseWriter) WriteHeader(code int) {
	s.statusCode = code
	s.ResponseWriter.WriteHeader(code)
}
