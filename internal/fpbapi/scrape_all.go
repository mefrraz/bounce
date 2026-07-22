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

	totalClubs := len(allClubs)
	log.Printf("\033[1;36m[scrape]\033[0m \033[33m%s\033[0m — \033[1m%d\033[0m clubs", season, totalClubs)
	start := time.Now()
	parallel := 5

	var total int64
	var errors int64
	var processed int64
	sem := make(chan struct{}, parallel)

	// Progress reporter
	stopProgress := make(chan struct{})
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				p := atomic.LoadInt64(&processed)
				t := atomic.LoadInt64(&total)
				e := atomic.LoadInt64(&errors)
				if p == 0 {
					continue
				}
				pct := int(p) * 100 / totalClubs
				elapsed := time.Since(start)
				eta := time.Duration(0)
				if p > 0 {
					eta = elapsed * time.Duration(totalClubs-int(p)) / time.Duration(p)
				}
				bar := progressBar(pct, 20)
				log.Printf("\033[36m[scrape]\033[0m %s \033[33m%s\033[0m %s \033[90m%3d%%\033[0m · %d/%d clubs · %d games · %d errs · ETA %v",
					bar, season, barSuffix(pct), pct, p, totalClubs, t, e, eta.Round(time.Second))
			case <-stopProgress:
				return
			}
		}
	}()

	for _, club := range allClubs {
		sem <- struct{}{}
		go func(clubID int) {
			defer func() { <-sem }()
			games, err := f.GetGamesByClub(fmt.Sprint(clubID), season, "", "")
			if err != nil {
				atomic.AddInt64(&errors, 1)
				atomic.AddInt64(&processed, 1)
				return
			}
			atomic.AddInt64(&total, int64(len(games)))
			atomic.AddInt64(&processed, 1)
		}(club.ID)
	}

	// Wait for all to finish
	for i := 0; i < parallel; i++ {
		sem <- struct{}{}
	}
	close(stopProgress)

	elapsed := time.Since(start).Round(time.Second)
	log.Printf("\033[1;32m[scrape]\033[0m \033[33m%s\033[0m \033[1;32m✓ done\033[0m: %d games, %d errors in %v",
		season, total, errors, elapsed)
}

func progressBar(pct, width int) string {
	filled := pct * width / 100
	bar := "\033[42m" // green background
	for i := 0; i < width; i++ {
		if i < filled {
			bar += " "
		} else {
			bar += "\033[0m\033[100m " // dark gray background
		}
	}
	bar += "\033[0m"
	return bar
}

func barSuffix(pct int) string {
	if pct < 100 {
		return ""
	}
	return ""
}
