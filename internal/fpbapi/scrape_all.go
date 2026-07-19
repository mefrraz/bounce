package fpbapi

import (
	"fmt"
	"log"
	"time"

	"github.com/mefrraz/bounce/internal/clubs"
)

// ScrapeAllClubs fetches games for all clubs for a season and persists to SQLite.
// Used by the daily cron to ensure the games table is complete for ELO calculation.
func (f *FPBAPI) ScrapeAllClubs(season string) {
	allClubs := clubs.All()
	if len(allClubs) == 0 {
		log.Printf("[scrape] no clubs loaded, skipping")
		return
	}

	log.Printf("[scrape] starting full scrape: %d clubs for %s", len(allClubs), season)
	start := time.Now()
	total := 0
	errors := 0

	// Process in batches of 5 for parallelism
	batchSize := 5
	for i := 0; i < len(allClubs); i += batchSize {
		end := i + batchSize
		if end > len(allClubs) { end = len(allClubs) }
		batch := allClubs[i:end]

		sem := make(chan struct{}, batchSize)
		results := make(chan int, len(batch))

		for _, club := range batch {
			sem <- struct{}{}
			go func(clubID int) {
				defer func() { <-sem }()
				games, err := f.GetGamesByClub(fmt.Sprint(clubID), season, "Senior", "masculino")
				if err != nil {
					log.Printf("[scrape] club %d error: %v", clubID, err)
					errors++
					results <- 0
					return
				}
				results <- len(games)
			}(club.ID)
		}

		for range batch {
			total += <-results
		}
	}

	elapsed := time.Since(start).Round(time.Second)
	log.Printf("[scrape] done: %d games, %d errors in %v", total, errors, elapsed)
}
