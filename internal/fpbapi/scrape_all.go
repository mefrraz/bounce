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
	parallel := 50

	var total int64
	var errors int64
	var processed int64
	var wg sync.WaitGroup
	sem := make(chan struct{}, parallel)

	for _, club := range allClubs {
		wg.Add(1)
		sem <- struct{}{}
		go func(clubID int) {
			defer func() { <-sem; wg.Done() }()
			games, err := f.GetGamesByClub(fmt.Sprint(clubID), season, "", "")
			if err != nil {
				atomic.AddInt64(&errors, 1)
				return
			}
			atomic.AddInt64(&total, int64(len(games)))
			n := int(atomic.AddInt64(&processed, 1))
			if n%max(1, len(allClubs)/10) == 0 {
				elapsed := time.Since(start).Round(time.Second)
				pct := n * 100 / len(allClubs)
				log.Printf("[scrape]   %d/%d clubs (%d%%, %v elapsed)", n, len(allClubs), pct, elapsed)
			}
		}(club.ID)
	}

	wg.Wait()
	elapsed := time.Since(start).Round(time.Second)
	log.Printf("[scrape] %s done: %d games, %d errors in %v", season, total, errors, elapsed)
}
