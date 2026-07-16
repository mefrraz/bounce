package main

import (
	"net/http"
	"sync/atomic"
	"time"
)

var (
	requestsTotal   uint64
	cacheHitsTotal  uint64
	cacheMissesTotal uint64
	fpbRequestsTotal uint64
	rateLimitedTotal uint64
)

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&requestsTotal, 1)
		t0 := time.Now()
		next.ServeHTTP(w, r)
		_ = time.Since(t0) // could log duration
	})
}

type metricsResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *metricsResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func apiKeyMiddleware(next http.Handler) http.Handler {
	key := apiKey
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if key != "" && r.Header.Get("X-API-Key") != key {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"invalid API key","code":"UNAUTHORIZED","status":401}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
