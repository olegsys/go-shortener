package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type loggerResponseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (r *loggerResponseWriter) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *loggerResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.size += size
	return size, err
}

func Logging(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			lw := &loggerResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(lw, r)
			duration := time.Since(start)
			logger.Info("HTTP request",
				zap.String("method", r.Method),
				zap.String("url", r.RequestURI),
				zap.Int("status_code", lw.statusCode),
				zap.Int("size", lw.size),
				zap.Duration("duration", duration),
			)
		})
	}
}
