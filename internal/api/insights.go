package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
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
	writeJSON(w, http.StatusOK, map[string]string{"message": "ELO ranking endpoint"})
}

func (h *InsightsHandler) GetClubELO(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"club_id": chi.URLParam(r, "clubID"), "message": "Club ELO history"})
}

func (h *InsightsHandler) GetPrediction(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"game_id": chi.URLParam(r, "gameID"), "message": "Game prediction"})
}

func (h *InsightsHandler) GetHeadToHead(w http.ResponseWriter, r *http.Request) {
	teamA := r.URL.Query().Get("team_a")
	teamB := r.URL.Query().Get("team_b")
	if teamA == "" || teamB == "" {
		http.Error(w, `{"error":"team_a and team_b required"}`, http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"team_a": teamA, "team_b": teamB, "message": "H2H history"})
}
