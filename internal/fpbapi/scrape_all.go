package fpbapi

import (
	"fmt"
	"log"
	"time"

	"github.com/mefrraz/bounce/internal/clubs"
)

// ScrapeAllClubs fetches games for all clubs for a season across ALL categories/genders.
func (f *FPBAPI) ScrapeAllClubs(season string) {
	allClubs := clubs.All()
	if len(allClubs) == 0 {
		log.Printf("[scrape] no clubs loaded, skipping")
		return
	}

	categories := []struct{ cat, gen string }{
		{"Senior", "masculino"}, {"Senior", "feminino"},
		{"Sub-18", "masculino"}, {"Sub-18", "feminino"},
		{"Sub-16", "masculino"}, {"Sub-16", "feminino"},
		{"Sub-14", "masculino"}, {"Sub-14", "feminino"},
	}

	log.Printf("[scrape] %s — %d clubs × %d categories", season, len(allClubs), len(categories))
	start := time.Now()
	total := 0
	errors := 0

	for _, cg := range categories {
		catStart := time.Now()
		catTotal := 0
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
					games, err := f.GetGamesByClub(fmt.Sprint(clubID), season, cg.cat, cg.gen)
					if err != nil { errors++; results <- 0; return }
					results <- len(games)
				}(club.ID)
			}
			for range batch { catTotal += <-results }
		}
		total += catTotal
		log.Printf("[scrape]   %s/%s: %d games in %v", cg.cat, cg.gen, catTotal, time.Since(catStart).Round(time.Second))
	}

	elapsed := time.Since(start).Round(time.Second)
	log.Printf("[scrape] %s done: %d games, %d errors in %v", season, total, errors, elapsed)

	// Discover categories after each season completes
	db := f.Cache().DB()
	var catCount int
	db.QueryRow("SELECT COUNT(DISTINCT escalao) FROM games").Scan(&catCount)
	if catCount > len(categories) {
		rows, _ := db.Query("SELECT DISTINCT escalao FROM games ORDER BY escalao")
		if rows != nil {
			var cats []string
			for rows.Next() { var c string; rows.Scan(&c); cats = append(cats, c) }
			rows.Close()
			log.Printf("[scrape] ⚡ discovered %d categories: %v", len(cats), cats)
		}
	}
}
