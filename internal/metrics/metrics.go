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

func StartRecording() {
	go func() {
		for {
			time.Sleep(60 * time.Second)
			RecordSnapshot()
		}
	}()
}
