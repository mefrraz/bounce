package metrics

import (
	"sync/atomic"
	"time"
)

var (
	RequestsTotal    uint64
	CacheHitsTotal   uint64
	CacheMissesTotal uint64
	FPBRequestsTotal uint64
	RateLimitedTotal uint64
)

func IncRequests()    { atomic.AddUint64(&RequestsTotal, 1) }
func IncCacheHit()    { atomic.AddUint64(&CacheHitsTotal, 1) }
func IncCacheMiss()   { atomic.AddUint64(&CacheMissesTotal, 1) }
func IncFPBRequest()  { atomic.AddUint64(&FPBRequestsTotal, 1) }
func IncRateLimited() { atomic.AddUint64(&RateLimitedTotal, 1) }
func ResetAll() {
	atomic.StoreUint64(&RequestsTotal, 0)
	atomic.StoreUint64(&CacheHitsTotal, 0)
	atomic.StoreUint64(&CacheMissesTotal, 0)
	atomic.StoreUint64(&FPBRequestsTotal, 0)
	atomic.StoreUint64(&RateLimitedTotal, 0)
	historyMu.Lock()
	history = nil
	historyMu.Unlock()
	if store != nil { store.PruneMetrics(time.Now()) }
	RecordSnapshot()
}

func StartRecording() {
	RecordSnapshot() // baseline at t=0 captures pre-warm activity
	go func() {
		for {
			time.Sleep(10 * time.Second)
			RecordSnapshot()
		}
	}()
}
