package fpbapi

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/mefrraz/bounce/internal/clubs"
)

// ScrapeAllClubs fetches all games for all clubs for a season.
func (f *FPBAPI) ScrapeAllClubs(season string) {
	allClubs := clubs.All()
	if len(allClubs) == 0 {
		log.Printf("[scrape] no clubs loaded, skipping")
		return
	}

	log.Printf("[scrape] %s — %d clubs (all categories, no filter)", season, len(allClubs))
	start := time.Now()
	parallel := 5

	var total int64
	var errors int64
	var processed int64
	sem := make(chan struct{}, parallel)

	for _, club := range allClubs {
		sem <- struct{}{}
		go func(clubID int) {
			defer func() { <-sem }()
			games, err := f.GetGamesByClub(fmt.Sprint(clubID), season, "", "")
			if err != nil {
				atomic.AddInt64(&errors, 1)
				return
			}
			atomic.AddInt64(&total, int64(len(games)))
			atomic.AddInt64(&processed, 1)
		}(club.ID)
	}

	// Wait for all to finish
	for i := 0; i < parallel; i++ { sem <- struct{}{} }

	elapsed := time.Since(start).Round(time.Second)
	log.Printf("[scrape] %s done: %d games, %d errors in %v", season, total, errors, elapsed)
}
