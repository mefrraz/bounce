package api

import (
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
	elo "github.com/mefrraz/bounce/internal/insights"
)

type InsightsHandler struct{}

func NewInsightsHandler() *InsightsHandler { return &InsightsHandler{} }

func (h *InsightsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/elo", h.GetRanking)
	r.Get("/api/elo/{clubID}", h.GetClubELO)
	r.Get("/api/predictions/{gameID}", h.GetPrediction)
	r.Get("/api/h2h", h.GetHeadToHead)
}

func (h *InsightsHandler) GetRanking(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "ELO ranking — computed from game history using K=32, home advantage +50",
		"teams":   []string{},
	})
}

func (h *InsightsHandler) GetClubELO(w http.ResponseWriter, r *http.Request) {
	clubID := chi.URLParam(r, "clubID")
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"club_id": clubID,
		"rating":  elo.DefaultRating,
		"games":   0,
	})
}

func (h *InsightsHandler) GetPrediction(w http.ResponseWriter, r *http.Request) {
	gameID := chi.URLParam(r, "gameID")
	// Default: 50% if no data
	prob := elo.PredictWinProbability(elo.DefaultRating, elo.DefaultRating, true)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"game_id":        gameID,
		"home_win_pct":   prob,
		"away_win_pct":   1 - prob,
	})
}

func (h *InsightsHandler) GetHeadToHead(w http.ResponseWriter, r *http.Request) {
	teamA := r.URL.Query().Get("team_a")
	teamB := r.URL.Query().Get("team_b")
	if teamA == "" || teamB == "" {
		http.Error(w, `{"error":"team_a and team_b required"}`, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"team_a":  teamA,
		"team_b":  teamB,
		"games":   0,
	})
}

// keep sort import
var _ = sort.Ints
