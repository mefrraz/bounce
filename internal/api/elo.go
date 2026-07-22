package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mefrraz/bounce/internal/elo"
)

// GetELO returns ELO ratings for a season from the SQLite elo_history table.
func (h *Handler) GetELO(w http.ResponseWriter, r *http.Request) {
	season := r.URL.Query().Get("season")
	if season == "" { season = CurrentSeason() }
	store := elo.NewStore(h.FPB.Cache().DB())
	ratings, err := store.GetSeason(season)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if ratings == nil { ratings = []elo.RatingRow{} }
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"season":  season,
		"ratings": ratings,
	})
}

// AdminELORecalculate starts an async ELO recalculation for a season.
func (h *Handler) AdminELORecalculate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	season := r.URL.Query().Get("season")
	if season == "" { season = CurrentSeason() }
	go func() {
		log.Printf("[elo] recalculate %s starting", season)
		if err := h.FPB.RecalculateELO(season); err != nil {
			log.Printf("[elo] recalculate %s error: %v", season, err)
		} else {
			log.Printf("[elo] recalculate %s done", season)
		}
	}()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(202)
	json.NewEncoder(w).Encode(map[string]string{"status": "started", "season": season})
}

// StartDailyScrapeAndELO runs the daily scraper + ELO at 3am.
func (h *Handler) StartDailyScrapeAndELO() {
	season := CurrentSeason()
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
			if now.After(next) { next = next.Add(24 * time.Hour) }
			log.Printf("[daily] next scrape in %v", next.Sub(now).Round(time.Minute))
			time.Sleep(next.Sub(now))

			log.Printf("[daily] scraping all clubs for %s", season)
			h.FPB.ScrapeAllClubs(season)
			log.Printf("[daily] recalculating ELO for %s", season)
			if err := h.FPB.RecalculateELO(season); err != nil {
				log.Printf("[daily] ELO error: %v", err)
			}
		}
	}()
}

// CurrentSeason returns e.g. "2025/2026".
func CurrentSeason() string {
	now := time.Now()
	year := now.Year()
	if now.Month() < 7 { year-- }
	return fmt.Sprintf("%d/%d", year, year+1)
}

// AdminTestConcurrency tests different concurrency levels against FPB.
func (h *Handler) AdminTestConcurrency(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	if flusher, ok := w.(http.Flusher); ok { flusher.Flush() }

	levels := []int{1, 2, 3, 5, 8, 10, 15, 20}
	results := make(map[string]interface{})
	season := r.URL.Query().Get("season")
	if season == "" { season = "2024/2025" }

	for _, n := range levels {
		log.Printf("[test] testing %d parallel...", n)
		start := time.Now()
		var ok, fail int64
		var wg sync.WaitGroup
		sem := make(chan struct{}, n)

		for i := 0; i < n*3; i++ {
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer func() { <-sem; wg.Done() }()
				_, err := h.FPB.GetGamesByClub("120", season, "", "")
				if err != nil { atomic.AddInt64(&fail, 1)
				} else { atomic.AddInt64(&ok, 1) }
			}()
		}
		for i := 0; i < n; i++ { sem <- struct{}{} }
		wg.Wait()
		elapsed := time.Since(start).Round(time.Millisecond)
		log.Printf("[test] %d parallel: %d ok, %d fail in %v", n, ok, fail, elapsed)
		results[fmt.Sprintf("parallel_%d", n)] = map[string]interface{}{
			"ok": ok, "fail": fail, "elapsed": elapsed.String(),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"season": season, "results": results,
	})
}
