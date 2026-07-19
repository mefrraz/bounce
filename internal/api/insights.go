package api

import (
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	elo "github.com/mefrraz/bounce/internal/insights"
)

type InsightsHandler struct {
	ratings map[string]*elo.Rating
	mu      sync.RWMutex
}

func NewInsightsHandler() *InsightsHandler {
	h := &InsightsHandler{ratings: make(map[string]*elo.Rating)}
	// Initialize with known top clubs
	for _, r := range []struct{ id, name string; rating float64 }{
		{"127", "SL Benfica", 1750},
		{"120", "FC Porto", 1680},
		{"169", "Sporting CP", 1650},
		{"119", "FC Gaia", 1500},
	} {
		h.ratings[r.id] = &elo.Rating{TeamID: r.id, Team: r.name, Rating: r.rating}
	}
	return h
}

func (h *InsightsHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/predictions/{gameID}", h.GetPrediction)
	r.Get("/api/h2h", h.GetHeadToHead)
}

func (h *InsightsHandler) GetRanking(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ratings := make([]elo.Rating, 0, len(h.ratings))
	for _, rt := range h.ratings { ratings = append(ratings, *rt) }
	writeJSON(w, http.StatusOK, ratings)
}

func (h *InsightsHandler) GetClubELO(w http.ResponseWriter, r *http.Request) {
	clubID := chi.URLParam(r, "clubID")
	h.mu.RLock()
	rt, ok := h.ratings[clubID]
	h.mu.RUnlock()
	if !ok { rt = &elo.Rating{TeamID: clubID, Rating: elo.DefaultRating} }
	writeJSON(w, http.StatusOK, rt)
}

func (h *InsightsHandler) GetPrediction(w http.ResponseWriter, r *http.Request) {
	gameID := chi.URLParam(r, "gameID")
	_ = gameID
	prob := elo.PredictWinProbability(elo.DefaultRating, elo.DefaultRating, true)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"game_id": gameID, "home_win_pct": prob, "away_win_pct": 1 - prob,
	})
}

func (h *InsightsHandler) GetHeadToHead(w http.ResponseWriter, r *http.Request) {
	teamA := r.URL.Query().Get("team_a")
	teamB := r.URL.Query().Get("team_b")
	if teamA == "" || teamB == "" {
		http.Error(w, `{"error":"team_a and team_b required"}`, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"team_a": teamA, "team_b": teamB, "games": 0})
}
