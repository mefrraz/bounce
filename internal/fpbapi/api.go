package fpbapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/mefrraz/bounce/internal/browser"
	"github.com/mefrraz/bounce/internal/cache"
	"github.com/mefrraz/bounce/internal/httpclient"
	"github.com/mefrraz/bounce/internal/models"
	"github.com/mefrraz/bounce/internal/scraper"
)

const fpbBase = "https://www.fpb.pt"

type FPBAPI struct {
	http    *httpclient.Client
	cache   *cache.Store
	browser *browser.Client
}

func New(c *httpclient.Client, s *cache.Store, b *browser.Client) *FPBAPI {
	return &FPBAPI{http: c, cache: s, browser: b}
}

// GetGame fetches a game by internal FPB ID via HTML scraping.
func (f *FPBAPI) GetGame(internalID string) (*models.GameDetail, error) {
	key := cache.CacheKey("game", internalID)
	if raw, ok := f.cache.Get(key); ok {
		var g models.GameDetail
		if err := json.Unmarshal(raw, &g); err == nil {
			log.Printf("[cache hit] game %s", internalID)
			return &g, nil
		}
	}

	u := fmt.Sprintf("%s/ficha-de-jogo?internalID=%s", fpbBase, url.PathEscape(internalID))
	log.Printf("[fetch] game %s from %s", internalID, u)

	body, err := f.http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("fetch game %s: %w", internalID, err)
	}

	html := string(body)

	// Check if we got a valid HTML page
	if strings.Contains(html, "403 Forbidden") || strings.Contains(html, "404") {
		return nil, fmt.Errorf("game %s not found on FPB", internalID)
	}

	detail, err := scraper.ScrapeGameDetail(html)
	if err != nil {
		return nil, fmt.Errorf("parse game %s: %w", internalID, err)
	}

	// If no scores and browser available, try JS rendering
	if detail.HomeScore == nil && f.browser != nil {
		log.Printf("[browser] JS render for game %s", internalID)
		jsHTML, err := f.browser.FetchHTML(u, "")
		if err == nil && jsHTML != "" {
			if jsDetail, err := scraper.ScrapeGameDetail(jsHTML); err == nil && jsDetail.HomeScore != nil {
				detail.HomeScore = jsDetail.HomeScore
				detail.AwayScore = jsDetail.AwayScore
				detail.Periods = jsDetail.Periods
				detail.HomeStats = jsDetail.HomeStats
				detail.AwayStats = jsDetail.AwayStats
				log.Printf("[browser] scores found for game %s: %d-%d", internalID, *detail.HomeScore, *detail.AwayScore)
			}
		}
	}

	detail.ID = internalID

	raw2, _ := json.Marshal(detail)
	f.cache.Set(key, raw2, cache.TTLRecent)
	return detail, nil
}

// GetGamesByClub fetches games for a club from FPB HTML.
func (f *FPBAPI) GetGamesByClub(clubID, season, category, gender string) ([]models.Game, error) {
	key := cache.CacheKey("games", "club", clubID, season, category, gender)
	if raw, ok := f.cache.Get(key); ok {
		var games []models.Game
		if err := json.Unmarshal(raw, &games); err == nil {
			return games, nil
		}
	}

	u := fmt.Sprintf("%s/calendario/clube_%s/?epoca=%s&escalao=%s&genero=%s",
		fpbBase, clubID, url.QueryEscape(season), url.QueryEscape(category), url.QueryEscape(gender))
	log.Printf("[fetch] club games from %s", u)

	body, err := f.http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("fetch club games: %w", err)
	}

	games := scraper.ScrapeGames(string(body), "AGENDADO")
	raw2, _ := json.Marshal(games)
	f.cache.Set(key, raw2, cache.TTLToday)
	return games, nil
}

// GetGamesByCompetition fetches games for a competition from FPB HTML (calendar or results).
func (f *FPBAPI) GetGamesByCompetition(compID, page string) ([]models.Game, error) {
	key := cache.CacheKey("games", "comp", compID, page)
	if raw, ok := f.cache.Get(key); ok {
		var games []models.Game
		if err := json.Unmarshal(raw, &games); err == nil {
			return games, nil
		}
	}

	u := fmt.Sprintf("%s/%s/%s", fpbBase, page, compID)
	log.Printf("[fetch] comp games from %s", u)

	body, err := f.http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("fetch comp games: %w", err)
	}

	status := "AGENDADO"
	if page == "resultados" {
		status = "FINALIZADO"
	}

	games := scraper.ScrapeGames(string(body), status)
	raw2, _ := json.Marshal(games)
	f.cache.Set(key, raw2, cache.TTLToday)
	return games, nil
}

// GetStandings fetches competition standings from FPB HTML + WordPress AJAX.
func (f *FPBAPI) GetStandings(compID string) ([]models.Standing, error) {
	key := cache.CacheKey("standings", compID)
	if raw, ok := f.cache.Get(key); ok {
		var s []models.Standing
		if err := json.Unmarshal(raw, &s); err == nil {
			return s, nil
		}
	}

	// First, get classification page to extract fase IDs
	classURL := fmt.Sprintf("%s/classificacao/%s", fpbBase, compID)
	body, err := f.http.Get(classURL)
	if err != nil {
		return nil, fmt.Errorf("fetch classification: %w", err)
	}

	fases := scraper.ExtractFaseIDs(string(body))
	if len(fases) == 0 {
		fases = append(fases, struct{ ID, Name string }{"30969", "Fase Regular"})
	}

	// Fetch standings for each fase via WordPress AJAX
	var allStandings []models.Standing
	for _, fase := range fases {
		ajaxKey := cache.CacheKey("standings", compID, fase.ID)
		var standings []models.Standing

		if raw, ok := f.cache.Get(ajaxKey); ok {
			json.Unmarshal(raw, &standings)
		} else {
			ajaxURL := fmt.Sprintf("%s/wp-admin/admin-ajax.php?action=get_more_fase_regular&competicao%%5B%%5D=%s&fase=%s",
				fpbBase, compID, fase.ID)
			log.Printf("[fetch] standings fase %s from %s", fase.ID, ajaxURL)

			ajaxBody, err := f.http.Get(ajaxURL)
			if err != nil {
				log.Printf("  warn: fase %s fetch failed: %v", fase.ID, err)
				continue
			}

			// WordPress AJAX returns JSON with a "result.body" field containing HTML
			var ajaxResp struct {
				Result struct {
					Body string `json:"body"`
				} `json:"result"`
			}
			if err := json.Unmarshal(ajaxBody, &ajaxResp); err == nil && ajaxResp.Result.Body != "" {
				standings = scraper.ScrapeStandings(ajaxResp.Result.Body)
			} else {
				// Fallback: parse direct HTML
				standings = scraper.ScrapeStandings(string(ajaxBody))
			}

			raw2, _ := json.Marshal(standings)
			f.cache.Set(ajaxKey, raw2, cache.TTLStandings)
		}

		allStandings = append(allStandings, standings...)
	}

	raw2, _ := json.Marshal(allStandings)
	f.cache.Set(key, raw2, cache.TTLStandings)
	return allStandings, nil
}

// GetCompetitions returns a hardcoded list of known competitions.
// The FPB doesn't have a public competition list API — these are maintained manually.
func (f *FPBAPI) GetCompetitions() ([]models.Competition, error) {
	// Hardcoded list matching the Dribly known competitions
	comps := []models.Competition{
		{ID: "12345", Name: "Liga Betclic", Abbreviation: "Betclic", Category: "Senior", Season: "2025/2026"},
		{ID: "12346", Name: "Proliga", Abbreviation: "Proliga", Category: "Senior", Season: "2025/2026"},
		{ID: "12347", Name: "1a Divisao", Abbreviation: "1a Div.", Category: "Senior", Season: "2025/2026"},
	}
	return comps, nil
}

// GetAthlete fetches and parses an athlete profile from FPB.
func (f *FPBAPI) GetAthlete(id string) (*scraper.AthleteData, error) {
	key := cache.CacheKey("athlete", id)
	if raw, ok := f.cache.Get(key); ok {
		var a scraper.AthleteData
		if err := json.Unmarshal(raw, &a); err == nil {
			return &a, nil
		}
	}
	u := fmt.Sprintf("%s/atletas/%s/", fpbBase, url.PathEscape(id))
	body, err := f.http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("fetch athlete: %w", err)
	}
	a := scraper.ScrapeAthlete(string(body))
	if a == nil {
		return nil, fmt.Errorf("failed to parse athlete %s", id)
	}
	raw2, _ := json.Marshal(a)
	f.cache.Set(key, raw2, cache.TTLHistorical)
	return a, nil
}

// GetTeam fetches and parses a team detail page from FPB.
func (f *FPBAPI) GetTeam(id string) (*scraper.TeamDetail, error) {
	key := cache.CacheKey("team", id)
	if raw, ok := f.cache.Get(key); ok {
		var td scraper.TeamDetail
		if err := json.Unmarshal(raw, &td); err == nil {
			return &td, nil
		}
	}
	u := fmt.Sprintf("%s/equipa/%s/", fpbBase, url.PathEscape(id))
	body, err := f.http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("fetch team: %w", err)
	}
	td := scraper.ScrapeTeamDetail(string(body))
	if td == nil {
		return nil, fmt.Errorf("failed to parse team %s", id)
	}
	raw2, _ := json.Marshal(td)
	f.cache.Set(key, raw2, cache.TTLHistorical)
	return td, nil
}

// GetClubTeams fetches the teams list for a club from FPB.
func (f *FPBAPI) GetClubTeams(clubID string) ([]models.Team, error) {
	key := cache.CacheKey("clubteams", clubID)
	if raw, ok := f.cache.Get(key); ok {
		var teams []models.Team
		if err := json.Unmarshal(raw, &teams); err == nil {
			return teams, nil
		}
	}
	u := fmt.Sprintf("%s/equipas/clube_%s/", fpbBase, clubID)
	body, err := f.http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("fetch club teams: %w", err)
	}
	teams := scraper.ScrapeClubTeams(string(body))
	raw2, _ := json.Marshal(teams)
	f.cache.Set(key, raw2, cache.TTLHistorical)
	return teams, nil
}

// GetTugaBasketStandings fetches standings from TugaBasket.
func (f *FPBAPI) GetTugaBasketStandings(competitionID string) ([]scraper.TugaBasketStanding, error) {
	key := cache.CacheKey("tugabasket", competitionID)
	if raw, ok := f.cache.Get(key); ok {
		var s []scraper.TugaBasketStanding
		if err := json.Unmarshal(raw, &s); err == nil {
			return s, nil
		}
	}
	u := fmt.Sprintf("https://resultados.tugabasket.com/getCompetitionDetails?competitionId=%s", competitionID)
	body, err := f.http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("fetch tugabasket: %w", err)
	}
	standings := scraper.ScrapeTugaBasketStandings(string(body))
	raw2, _ := json.Marshal(standings)
	f.cache.Set(key, raw2, cache.TTLStandings)
	return standings, nil
}
