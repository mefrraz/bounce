package metrics

import (
	"runtime"
	"sync"
	"time"
)

type Snapshot struct {
	Time         time.Time
	Requests     uint64
	CacheHits    uint64
	CacheMisses  uint64
	FPBRequests  uint64
	RateLimited  uint64
	MemAlloc     uint64
	Goroutines   int
}

var (
	history   []Snapshot
	historyMu sync.Mutex
)

func RecordSnapshot() {
	historyMu.Lock()
	defer historyMu.Unlock()
	history = append(history, Snapshot{
		Time:        time.Now(),
		Requests:    RequestsTotal,
		CacheHits:   CacheHitsTotal,
		CacheMisses: CacheMissesTotal,
		FPBRequests: FPBRequestsTotal,
		RateLimited: RateLimitedTotal,
		Goroutines:  runtime.NumGoroutine(),
	})
	// Keep only last 7 days (10080 minutes)
	if len(history) > 10080 {
		history = history[len(history)-10080:]
	}
}

func GetHistory(since time.Duration) []Snapshot {
	historyMu.Lock()
	defer historyMu.Unlock()
	cutoff := time.Now().Add(-since)
	var result []Snapshot
	for _, s := range history {
		if s.Time.After(cutoff) { result = append(result, s) }
	}
	return result
}
