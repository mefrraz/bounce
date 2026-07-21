package fpbapi

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mefrraz/bounce/internal/clubs"
)

// ScrapeAllClubs fetches all games for all clubs for a season (no category filter — gets everything).
func (f *FPBAPI) ScrapeAllClubs(season string) {
	allClubs := clubs.All()
	if len(allClubs) == 0 {
		log.Printf("[scrape] no clubs loaded, skipping")
		return
	}

	log.Printf("[scrape] %s — %d clubs (all categories, no filter)", season, len(allClubs))
	start := time.Now()
	parallel := 4
	
	var total int64
	var errors int64
	var processed int64
	var started int64
	var wg sync.WaitGroup
	sem := make(chan struct{}, parallel)

	for i, club := range allClubs {
		wg.Add(1)
		sem <- struct{}{}
		atomic.AddInt64(&started, 1)
		go func(clubID int, idx int64) {
			defer func() { <-sem; wg.Done() }()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[scrape] club %d PANIC: %v", clubID, r)
					atomic.AddInt64(&errors, 1)
					atomic.AddInt64(&processed, 1)
				}
			}()
			games, err := f.GetGamesByClub(fmt.Sprint(clubID), season, "", "")
			if err != nil {
				atomic.AddInt64(&errors, 1)
				if errors <= 3 {
					log.Printf("[scrape] club %d error: %v", clubID, err)
				}
			} else {
				atomic.AddInt64(&total, int64(len(games)))
			}
			n := int(atomic.AddInt64(&processed, 1))
			elapsed := time.Since(start).Round(time.Second)
			pct := n * 100 / len(allClubs)
			inFlight := atomic.LoadInt64(&started) - atomic.LoadInt64(&processed) - atomic.LoadInt64(&errors)
			if n%max(1, len(allClubs)/10) == 0 {
				log.Printf("[scrape]   %d/%d clubs (%d%%, %v elapsed, %d err, %d in flight)", n, len(allClubs), pct, elapsed, errors, inFlight)
			}
		}(club.ID, int64(i+1))
	}

	wg.Wait()
	elapsed := time.Since(start).Round(time.Second)
	log.Printf("[scrape] %s done: %d games, %d errors in %v", season, total, errors, elapsed)
}
