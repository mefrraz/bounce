package api

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
}

func (h *Handler) GetGames(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	competition := r.URL.Query().Get("competition")
	if date == "" {
		http.Error(w, `{"error":"date required"}`, http.StatusBadRequest)
		return
	}
	if competition == "" {
		writeJSON(w, http.StatusOK, []interface{}{})
		return
	}
	games, err := h.FPB.GetGames(competition, date)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, games)
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

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
