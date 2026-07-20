package fpbapi

import (
	"fmt"
	"log"
	"sort"
	"sync/atomic"
	"time"

	"github.com/mefrraz/bounce/internal/clubs"
)

// ScrapeAllClubs fetches games for ALL clubs for a season, auto-discovering categories first.
func (f *FPBAPI) ScrapeAllClubs(season string) {
	allClubs := clubs.All()
	if len(allClubs) == 0 {
		log.Printf("[scrape] no clubs loaded, skipping")
		return
	}

	// 1. Discover categories — fetch one big club without category filter
	categories := f.discoverCategories(season)
	log.Printf("[scrape] %s — %d clubs × %d categories (discovered)", season, len(allClubs), len(categories))

	start := time.Now()
	total := 0
	errors := 0
	parallel := 50

	for _, cat := range categories {
		catStart := time.Now()
		var catTotal int64
		sem := make(chan struct{}, parallel)

		for _, club := range allClubs {
			sem <- struct{}{}
			go func(clubID int, category string) {
				defer func() { <-sem }()
				games, err := f.GetGamesByClub(fmt.Sprint(clubID), season, category, "")
				if err != nil { errors++; return }
				atomic.AddInt64(&catTotal, int64(len(games)))
			}(club.ID, cat)
		}
		for i := 0; i < parallel; i++ { sem <- struct{}{} }

		total += int(catTotal)
		log.Printf("[scrape]   %s: %d games (%v)", cat, catTotal, time.Since(catStart).Round(time.Second))
	}

	elapsed := time.Since(start).Round(time.Second)
	log.Printf("[scrape] %s done: %d games, %d errors in %v", season, total, errors, elapsed)
}

// discoverCategories fetches a few big clubs without category filter to find all categories.
func (f *FPBAPI) discoverCategories(season string) []string {
	bigClubs := []int{120, 127, 169} // Porto, Benfica, Sporting
	seen := map[string]bool{}

	for _, clubID := range bigClubs {
		games, err := f.GetGamesByClub(fmt.Sprint(clubID), season, "", "")
		if err != nil || len(games) == 0 {
			continue
		}
		for _, g := range games {
			if g.Category != "" {
				seen[g.Category] = true
			}
		}
	}

	var result []string
	for cat := range seen { result = append(result, cat) }
	sort.Strings(result)

	if len(result) == 0 {
		return []string{"Senior", "Sub-18", "Sub-16", "Sub-14"}
	}

	log.Printf("[scrape] discovered %d categories: %v", len(result), result)
	return result
}
