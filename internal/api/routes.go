package api

// @title Bounce API
// @version 4.1.0
// @description Smart Sports Data Proxy for Portuguese basketball. Aggregates data from FPB and TugaBasket.
// @host localhost:3001
// @BasePath /

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mefrraz/bounce/internal/cache"
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
r.Get("/api/games/today", h.GetToday)
r.Get("/api/games/live", h.GetLive)
r.Get("/api/games/paginated", h.GetGamesPaginated)
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
// GetGames godoc
// @Summary      List club games with scores
// @Description  Returns all games for a club in a season from FPB.
// @Tags         games
// @Produce      json
// @Param        club    query     int     true  "Club ID"  example(119)
// @Param        season  query     string  true  "Season YYYY/YYYY"  example(2025/2026)
// @Success      200     {array}   models.Game
// @Router       /api/games [get]

// GetGame godoc
// @Summary      Game detail with scores
// @Description  Returns full game detail: teams, score, periods, logos.
// @Tags         games
// @Produce      json
// @Param        id   path      string  true  "Internal game ID"
// @Success      200  {object}  models.GameDetail
// @Router       /api/game/{id} [get]

func (h *Handler) GetGames(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	club := q.Get("club")
	competition := q.Get("competition")

	if club != "" {
		season := q.Get("season")
		if season == "" {
			season = cache.CurrentSeason()
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
		season := q.Get("season")
		if season == "" { season = cache.CurrentSeason() }
		games, err := h.FPB.GetGamesByCompetition(competition, season)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
		}
		writeJSON(w, http.StatusOK, games)
	return
	}

	writeJSON(w, http.StatusOK, []interface{}{})
}

// GetStandings godoc
// @Summary      Competition standings
// @Description  Returns the classification table for a competition.
// @Tags         competitions
// @Produce      json
// @Param        compID   path      string  true  "Competition ID"  example(10902)
// @Success      200      {array}   models.Standing
// @Router       /api/standings/{compID} [get]

func (h *Handler) GetStandings(w http.ResponseWriter, r *http.Request) {
	compID := chi.URLParam(r, "compID")
	standings, err := h.FPB.GetStandings(compID)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
	return
	}
	writeJSON(w, http.StatusOK, standings)
}

// GetGame godoc
// @Summary      Game detail with scores
// @Description  Returns full game detail: teams, score, periods, logos.
// @Tags         games
// @Produce      json
// @Param        id   path      string  true  "Internal game ID"
// @Success      200  {object}  models.GameDetail
// @Router       /api/game/{id} [get]

func (h *Handler) GetGame(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "internalID")
	game, err := h.FPB.GetGame(id)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
	return
	}
	writeJSON(w, http.StatusOK, game)
}

// GetCompetitions godoc
// @Summary      List competitions
// @Description  Returns known competitions from FPB.
// @Tags         competitions
// @Produce      json
// @Success      200  {array}   models.Competition
// @Router       /api/competitions [get]

func (h *Handler) GetCompetitions(w http.ResponseWriter, r *http.Request) {
	comps, err := h.FPB.GetCompetitions()
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
	return
	}
	writeJSON(w, http.StatusOK, comps)
}

// GetAthlete godoc
// @Summary      Athlete profile
// @Description  Returns athlete data: name, photo, position, club, stats.
// @Tags         athletes
// @Produce      json
// @Param        id   path      string  true  "Athlete ID"
// @Success      200  {object}  scraper.AthleteData
// @Router       /api/athlete/{id} [get]

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

// GetTugaBasketStandings godoc
// @Summary      TugaBasket standings
// @Description  Returns standings from TugaBasket for a competition.
// @Tags         tugabasket
// @Produce      json
// @Param        competitionId   query     string  true  "Competition ID"
// @Success      200             {array}   scraper.TugaBasketStanding
// @Router       /api/tugabasket/standings [get]

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

// GetTugaBasketPlayers godoc
// @Summary      TugaBasket player stats
// @Description  Returns individual player statistics (22 fields).
// @Tags         tugabasket
// @Produce      json
// @Param        competitionId   query     string  true  "Competition ID"
// @Success      200             {array}   scraper.TBPlayerStat
// @Router       /api/tugabasket/players [get]

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

// GetTugaBasketTeams godoc
// @Summary      TugaBasket team stats
// @Description  Returns aggregated team statistics.
// @Tags         tugabasket
// @Produce      json
// @Param        competitionId   query     string  true  "Competition ID"
// @Success      200             {array}   scraper.TBTeamStat
// @Router       /api/tugabasket/teams [get]

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

func jsonError(w http.ResponseWriter, msg, code string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg, "code": code, "status": fmt.Sprint(status)})
}

// GetToday returns today's games across all known competitions.
func (h *Handler) GetToday(w http.ResponseWriter, r *http.Request) {
		_ = time.Now()
	var all []interface{}
	for _, c := range []string{"10902","10903","10904","10906","10907"} {
		games, _ := h.FPB.GetGamesByCompetition(c, cache.CurrentSeason())
		for _, g := range games {
			if cache.IsToday(g.Date) { all = append(all, g) }
		}
	}
	if all == nil { all = []interface{}{} }; writeJSON(w, http.StatusOK, all)
}

// GetLive returns games currently in progress.
func (h *Handler) GetLive(w http.ResponseWriter, r *http.Request) {
	var live []interface{}
	for _, c := range []string{"10902","10903","10904","10906","10907"} {
		games, _ := h.FPB.GetGamesByCompetition(c, cache.CurrentSeason())
		for _, g := range games {
			if g.Status == "AO VIVO" || g.Status == "EM CURSO" { live = append(live, g) }
		}
	}
	if live == nil { live = []interface{}{} }; writeJSON(w, http.StatusOK, live)
}

// GetGamesPaginated handles ?club=ID with pagination.
func (h *Handler) GetGamesPaginated(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := atoiQ(q, "limit", 50)
	offset := atoiQ(q, "offset", 0)
	club := q.Get("club")
	season := q.Get("season")
	if season == "" { season = cache.CurrentSeason() }

	games, err := h.FPB.GetGamesByClub(club, season, "Senior", "masculino")
	if err != nil { jsonError(w, err.Error(), "FETCH_ERROR", 502); return }

	total := len(games)
	if offset >= total { offset = total }
	end := offset + limit
	if end > total { end = total }

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total": total, "limit": limit, "offset": offset,
		"games": games[offset:end],
	})
}

func atoiQ(q url.Values, key string, defaultVal int) int {
	s := q.Get(key)
	if s == "" { return defaultVal }
	v := 0
	for _, c := range s { v = v*10 + int(c-'0') }
return v
}
