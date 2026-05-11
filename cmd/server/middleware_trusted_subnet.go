package main

import (
	"net"
	"net/http"
)

// TrustedSubnetMiddleware проверяет, что IP адрес из заголовка X-Real-IP входит в доверенную подсеть.
// Если TrustedSubnet пуст, проверка пропускается.
// Если IP не входит в доверенную подсеть, возвращается 403 Forbidden.
func (s *Server) TrustedSubnetMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Если TrustedSubnet не установлен, пропускаем проверку
		if s.trustedSubnet == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Получаем IP адрес из заголовка X-Real-IP
		realIP := r.Header.Get("X-Real-IP")
		if realIP == "" {
			http.Error(w, "X-Real-IP header is required", http.StatusForbidden)
			return
		}

		// Парсим доверенную подсеть
		_, trustedNetwork, err := net.ParseCIDR(s.trustedSubnet)
		if err != nil {
			s.logger.Error("failed to parse trusted subnet", err.Error())
			http.Error(w, "server configuration error", http.StatusInternalServerError)
			return
		}

		// Парсим IP адрес из заголовка
		agentIP := net.ParseIP(realIP)
		if agentIP == nil {
			http.Error(w, "invalid X-Real-IP format", http.StatusForbidden)
			return
		}

		// Проверяем, входит ли IP в доверенную подсеть
		if !trustedNetwork.Contains(agentIP) {
			http.Error(w, "agent IP is not in trusted subnet", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
