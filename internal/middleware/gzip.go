package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"
)

// DecompressMiddleware распаковывает тело запроса, если оно сжато с помощью gzip
func DecompressMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Error decompressing body", http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = gz
		}
		next.ServeHTTP(w, r)
	})
}
