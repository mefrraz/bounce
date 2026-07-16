package api

// @title Bounce API
// @version 4.1.0
// @description Smart Sports Data Proxy for Portuguese basketball. Aggregates data from FPB and TugaBasket.
// @host localhost:3001
// @BasePath /

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mefrraz/bounce/internal/fpbapi"
)

type Handler struct {
	FPB *fpbapi.FPBAPI
}

func NewHandler(fpb *fpbapi.FPBAPI) *Handler {
	return &Handler{FPB: fpb}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/api/games", h.GetGames)
	r.Get("/api/standings/{compID}", h.GetStandings)
	r.Get("/api/game/{internalID}", h.GetGame)
	r.Get("/api/competitions", h.GetCompetitions)
	r.Get("/api/athlete/{id}", h.GetAthlete)
	r.Get("/api/team/{id}", h.GetTeam)
	r.Get("/api/club/{clubID}/teams", h.GetClubTeams)
	r.Get("/api/tugabasket/standings", h.GetTugaBasketStandings)
	r.Get("/api/tugabasket/players", h.GetTugaBasketPlayers)
	r.Get("/api/tugabasket/teams", h.GetTugaBasketTeams)
}

// GetGames supports:
//   ?club=ID&season=2025/2026&category=Senior&gender=masculino
//   ?competition=ID&page=calendario|resultados
func (h *Handler) GetGames(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	club := q.Get("club")
	competition := q.Get("competition")

	if club != "" {
		season := q.Get("season")
		if season == "" {
			season = "2025/2026"
		}
		category := q.Get("category")
		if category == "" {
			category = "Senior"
		}
		gender := q.Get("gender")
		if gender == "" {
			gender = "masculino"
		}
		games, err := h.FPB.GetGamesByClub(club, season, category, gender)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, games)
		return
	}

	if competition != "" {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}

	writeJSON(w, http.StatusOK, []interface{}{})
}

func (h *Handler) GetStandings(w http.ResponseWriter, r *http.Request) {
	compID := chi.URLParam(r, "compID")
	standings, err := h.FPB.GetStandings(compID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, standings)
}

func (h *Handler) GetGame(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "internalID")
	game, err := h.FPB.GetGame(id)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, game)
}

func (h *Handler) GetCompetitions(w http.ResponseWriter, r *http.Request) {
	comps, err := h.FPB.GetCompetitions()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, comps)
}

func (h *Handler) GetAthlete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	a, err := h.FPB.GetAthlete(id)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	td, err := h.FPB.GetTeam(id)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, td)
}

func (h *Handler) GetClubTeams(w http.ResponseWriter, r *http.Request) {
	clubID := chi.URLParam(r, "clubID")
	teams, err := h.FPB.GetClubTeams(clubID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, teams)
}

func (h *Handler) GetTugaBasketStandings(w http.ResponseWriter, r *http.Request) {
	compID := r.URL.Query().Get("competitionId")
	if compID == "" {
		http.Error(w, `{"error":"competitionId required"}`, http.StatusBadRequest)
		return
	}
	standings, err := h.FPB.GetTugaBasketStandings(compID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, standings)
}

func (h *Handler) GetTugaBasketPlayers(w http.ResponseWriter, r *http.Request) {
	compID := r.URL.Query().Get("competitionId")
	if compID == "" {
		http.Error(w, `{"error":"competitionId required"}`, http.StatusBadRequest)
		return
	}
	players, err := h.FPB.GetTugaBasketPlayers(compID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, players)
}

func (h *Handler) GetTugaBasketTeams(w http.ResponseWriter, r *http.Request) {
	compID := r.URL.Query().Get("competitionId")
	if compID == "" {
		http.Error(w, `{"error":"competitionId required"}`, http.StatusBadRequest)
		return
	}
	teams, err := h.FPB.GetTugaBasketTeams(compID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, teams)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
