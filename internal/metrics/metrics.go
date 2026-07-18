package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

var (
	RequestsTotal    uint64
	CacheHitsTotal   uint64
	CacheMissesTotal uint64
	FPBRequestsTotal uint64
	RateLimitedTotal uint64

	rateLastReq   uint64
	rateLastTime  time.Time
	rateMu        sync.Mutex
	rateLastValue float64
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
	rateMu.Lock()
	rateLastReq = 0
	rateLastTime = time.Time{}
	rateLastValue = 0
	rateMu.Unlock()
	historyMu.Lock()
	history = nil
	historyMu.Unlock()
	if store != nil { store.PruneMetrics(time.Now().Add(24 * time.Hour)) }
	RecordSnapshot()
}

func StartRecording() {
	rateLastTime = time.Now()
	rateLastReq = RequestsTotal
	RecordSnapshot() // baseline at t=0 captures pre-warm activity
	go func() {
		for {
			time.Sleep(10 * time.Second)
			RecordSnapshot()
		}
	}()
}

// ReqRate returns the current requests per second using a smoothed exponential moving average.
func ReqRate() float64 {
	now := time.Now()
	current := atomic.LoadUint64(&RequestsTotal)
	rateMu.Lock()
	defer rateMu.Unlock()
	if !rateLastTime.IsZero() {
		elapsed := now.Sub(rateLastTime).Seconds()
		if elapsed > 0 {
			instant := float64(current-rateLastReq) / elapsed
			if instant < 0 { instant = 0 }
			if rateLastValue == 0 {
				rateLastValue = instant
			} else {
				rateLastValue = 0.7*instant + 0.3*rateLastValue
			}
		}
	}
	rateLastReq = current
	rateLastTime = now
	return rateLastValue
}
