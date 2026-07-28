package middleware

import (
	"net"
	"net/http"
)

// TrustedSubnetMiddleware middleware, который проверяет,
// что IP-адрес из заголовка X-Real-IP входит в доверенную подсеть (CIDR).
// Если trustedSubnet пуст, доступ запрещён для любого запроса.
func TrustedSubnetMiddleware(trustedSubnet string) func(http.Handler) http.Handler {
	var ipNet *net.IPNet

	if trustedSubnet != "" {
		_, parsed, err := net.ParseCIDR(trustedSubnet)
		if err == nil {
			ipNet = parsed
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Если подсеть не задана или не распарсилась — запрещаем доступ
			if ipNet == nil {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			realIP := r.Header.Get("X-Real-IP")
			if realIP == "" {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			ip := net.ParseIP(realIP)
			if ip == nil {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			if !ipNet.Contains(ip) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
