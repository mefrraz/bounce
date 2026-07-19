package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/mefrraz/bounce/internal/elo"
)

// GetELO returns ELO ratings for a season from the SQLite elo_history table.
func (h *Handler) GetELO(w http.ResponseWriter, r *http.Request) {
	season := r.URL.Query().Get("season")
	if season == "" { season = "2025/2026" }
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
	if season == "" {
		http.Error(w, "missing season param", 400)
		return
	}
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
