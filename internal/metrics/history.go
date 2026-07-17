package metrics

import (
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/mefrraz/bounce/internal/cache"
)

type Snapshot struct {
	Time        time.Time `json:"time"`
	Requests    uint64    `json:"requests"`
	CacheHits   uint64    `json:"cache_hits"`
	CacheMisses uint64    `json:"cache_misses"`
	FPBRequests uint64    `json:"fpb_requests"`
	RateLimited uint64    `json:"rate_limited"`
	MemAlloc    uint64    `json:"-"`
	Goroutines  int       `json:"-"`
}

var (
	history   []Snapshot
	historyMu sync.Mutex
	store     *cache.Store
)

func SetStore(s *cache.Store) { store = s }

func LoadHistory() {
	if store == nil { return }
	rows := store.LoadMetrics(time.Now().Add(-7*24*time.Hour), 60480)
	historyMu.Lock()
	defer historyMu.Unlock()
	for _, r := range rows {
		history = append(history, Snapshot{
			Time: time.Unix(r.Time, 0), Requests: uint64(r.Requests),
			CacheHits: uint64(r.CacheHits), CacheMisses: uint64(r.CacheMisses),
			FPBRequests: uint64(r.FPBRequests), RateLimited: uint64(r.RateLimited),
			Goroutines: int(r.Goroutines),
		})
	}
	log.Printf("[metrics] loaded %d snapshots from db", len(history))
}

func RecordSnapshot() {
	historyMu.Lock()
	defer historyMu.Unlock()
	s := Snapshot{
		Time: time.Now(), Requests: RequestsTotal, CacheHits: CacheHitsTotal,
		CacheMisses: CacheMissesTotal, FPBRequests: FPBRequestsTotal,
		RateLimited: RateLimitedTotal, Goroutines: runtime.NumGoroutine(),
	}
	history = append(history, s)
	if len(history) > 60480 {
		history = history[len(history)-60480:]
	}
	if store != nil {
		store.SaveMetric(s.Time, s.Requests, s.CacheHits, s.CacheMisses, s.FPBRequests, s.RateLimited, s.Goroutines)
	}
	if len(history)%1000 == 0 && store != nil {
		store.PruneMetrics(time.Now().Add(-7 * 24 * time.Hour))
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
