package middleware

import (
	"net"
	"net/http"
	"os"
	"strings"
)

var allowedIPs []*net.IPNet
var allowedIPsLoaded bool

func loadAllowedIPs() {
	if allowedIPsLoaded {
		return
	}
	allowedIPsLoaded = true

	raw := strings.TrimSpace(os.Getenv("UNIVERSITY_IP"))
	if raw == "" {
		return
	}

	entries := strings.Split(raw, ",")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		if !strings.Contains(entry, "/") {
			if strings.Contains(entry, ":") {
				entry = entry + "/128"
			} else {
				entry = entry + "/32"
			}
		}

		_, cidr, err := net.ParseCIDR(entry)
		if err != nil {
			continue
		}
		allowedIPs = append(allowedIPs, cidr)
	}
}

func IPCheck(next http.Handler) http.Handler {
	loadAllowedIPs()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(allowedIPs) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		clientIP := extractClientIP(r)
		if clientIP == "" {
			http.Error(w, `{"error":"unable to determine client IP"}`, http.StatusForbidden)
			return
		}

		parsedIP := net.ParseIP(clientIP)
		if parsedIP == nil {
			http.Error(w, `{"error":"invalid client IP"}`, http.StatusForbidden)
			return
		}

		for _, cidr := range allowedIPs {
			if cidr.Contains(parsedIP) {
				next.ServeHTTP(w, r)
				return
			}
		}

		http.Error(w, `{"error":"access denied: not on the university network"}`, http.StatusForbidden)
	})
}

func extractClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return strings.TrimSpace(xri)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
