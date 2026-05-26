package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ipRateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*rateBucket
	maxReqs  int
	window   time.Duration
}

type rateBucket struct {
	count       int
	windowStart time.Time
}

func newIPRateLimiter(maxReqs int, window time.Duration) *ipRateLimiter {
	rl := &ipRateLimiter{
		visitors: make(map[string]*rateBucket),
		maxReqs:  maxReqs,
		window:   window,
	}
	go rl.periodicCleanup()
	return rl
}

func (rl *ipRateLimiter) periodicCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, bucket := range rl.visitors {
			if now.Sub(bucket.windowStart) > rl.window {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *ipRateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	bucket, exists := rl.visitors[ip]

	if !exists || now.Sub(bucket.windowStart) > rl.window {
		rl.visitors[ip] = &rateBucket{count: 1, windowStart: now}
		return true
	}

	if bucket.count >= rl.maxReqs {
		return false
	}

	bucket.count++
	return true
}

var loginRateLimiter = newIPRateLimiter(5, 10*time.Second)
var apiRateLimiter = newIPRateLimiter(60, 30*time.Second)

func LoginRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractClientIPForRL(r)
		if !loginRateLimiter.allow(ip) {
			http.Error(w, `{"error":"too many login attempts, try again later"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func GlobalRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractClientIPForRL(r)
		if !apiRateLimiter.allow(ip) {
			http.Error(w, `{"error":"too many requests, slow down"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func extractClientIPForRL(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		idx := strings.Index(xff, ",")
		if idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
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
