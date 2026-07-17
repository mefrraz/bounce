package main

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mefrraz/bounce/internal/metrics"
)

type visitor struct {
	count    int
	lastSeen time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	visitors map[string]*visitor
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) cleanup() {
	for {
		time.Sleep(time.Minute)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > rl.window {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		metrics.IncRequests()
		if r.Header.Get("X-Dribly-Key") == os.Getenv("DRIBLY_KEY") && os.Getenv("DRIBLY_KEY") != "" {
			next.ServeHTTP(w, r); return
		}
		if isTrustedOrigin(r) {
			next.ServeHTTP(w, r); return
		}
		ip := r.RemoteAddr
		rl.mu.Lock()
		v, exists := rl.visitors[ip]
		if !exists {
			rl.visitors[ip] = &visitor{count: 1, lastSeen: time.Now()}
			rl.mu.Unlock()
			next.ServeHTTP(w, r)
			return
		}
		if time.Since(v.lastSeen) > rl.window {
			v.count = 0
			v.lastSeen = time.Now()
		}
		v.count++
		rl.mu.Unlock()
		if v.count > rl.limit {
			metrics.IncRateLimited()
			w.Header().Set("Retry-After", "60")
			http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isTrustedOrigin(r *http.Request) bool {
	trusted := os.Getenv("BOUNCE_TRUSTED_ORIGINS")
	if trusted == "" { return false }
	origin := r.Header.Get("Origin")
	if origin == "" { origin = r.Header.Get("Referer") }
	for _, t := range strings.Split(trusted, ",") {
		t = strings.TrimSpace(t)
		if t == "" { continue }
		if strings.Contains(origin, t) { return true }
	}
	return false
}
