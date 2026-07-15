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
const sav2Base = "https://sav2.fpb.pt"

type FPBAPI struct {
	http    *httpclient.Client
	cache   *cache.Store
	browser *browser.Client
}

func New(c *httpclient.Client, s *cache.Store, b *browser.Client) *FPBAPI {
	return &FPBAPI{http: c, cache: s, browser: b}
}

func (f *FPBAPI) GetGame(internalID string) (*models.GameDetail, error) {
	key := cache.CacheKey("game", internalID)
	if raw, ok := f.cache.Get(key); ok {
		var g models.GameDetail
		if err := json.Unmarshal(raw, &g); err == nil { return &g, nil }
	}
	u := fmt.Sprintf("%s/ficha-de-jogo?internalID=%s", fpbBase, url.PathEscape(internalID))
	body, err := f.http.Get(u)
	if err != nil { return nil, fmt.Errorf("fetch game: %w", err) }
	detail, err := scraper.ScrapeGameDetail(string(body))
	if err != nil { return nil, fmt.Errorf("parse game: %w", err) }
	detail.ID = internalID
	raw2, _ := json.Marshal(detail)
	f.cache.Set(key, raw2, cache.TTLRecent)
	return detail, nil
}

func (f *FPBAPI) GetGamesByClub(clubID, season, category, gender string) ([]models.Game, error) {
	key := cache.CacheKey("games", "club", clubID, season)
	if raw, ok := f.cache.Get(key); ok {
		var games []models.Game
		if err := json.Unmarshal(raw, &games); err == nil { return games, nil }
	}

	parts := strings.Split(season, "/")
	if len(parts) != 2 { return nil, fmt.Errorf("invalid season: %s", season) }
	yearStart, yearEnd := parts[0], parts[1]

	params := url.Values{}
	params.Set("action", "get_more_days")
	params.Set("epoca", season)
	params.Set("escalao", "Senior")
	params.Set("genero", "masculino")
	params.Set("clube", clubID)
	params.Set("period[time_option]", "fromInit")
	params.Set("period[from_date]", yearStart+"/09/01")
	params.Set("period[to_date]", yearEnd+"/06/30")

	u := fpbBase + "/wp-admin/admin-ajax.php?" + params.Encode()
	log.Printf("[ajax] club %s season %s", clubID, season)

	body, err := f.http.Get(u)
	if err != nil { return nil, fmt.Errorf("fetch: %w", err) }

	var ajaxResp struct {
		Result  interface{} `json:"result"`
		Hasmore bool        `json:"hasmore"`
	}
	if err := json.Unmarshal(body, &ajaxResp); err != nil {
		games := scraper.ScrapeGames(string(body), "FINALIZADO")
		raw2, _ := json.Marshal(games)
		f.cache.Set(key, raw2, cache.TTLToday)
		return games, nil
	}

	var allHTML strings.Builder
	switch v := ajaxResp.Result.(type) {
	case string: allHTML.WriteString(v)
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok { allHTML.WriteString(s) }
		}
	}

	games := scraper.ScrapeGames(allHTML.String(), "FINALIZADO")
	log.Printf("[ajax] %d games for club %s season %s", len(games), clubID, season)
	raw2, _ := json.Marshal(games)
	f.cache.Set(key, raw2, cache.TTLToday)
	return games, nil
}

func (f *FPBAPI) GetStandings(compID string) ([]models.Standing, error) {
	key := cache.CacheKey("standings", compID)
	if raw, ok := f.cache.Get(key); ok {
		var s []models.Standing
		if err := json.Unmarshal(raw, &s); err == nil { return s, nil }
	}
	u := fmt.Sprintf("%s/api/classificacao/%s", sav2Base, compID)
	body, err := f.http.Get(u)
	if err != nil { return nil, fmt.Errorf("fetch standings: %w", err) }
	var standings []models.Standing
	if err := json.Unmarshal(body, &standings); err != nil {
		standings = scraper.ScrapeStandings(string(body))
	}
	raw2, _ := json.Marshal(standings)
	f.cache.Set(key, raw2, cache.TTLStandings)
	return standings, nil
}

func (f *FPBAPI) GetCompetitions() ([]models.Competition, error) {
	return []models.Competition{
		{ID: "10902", Name: "Liga Betclic Masculina", Abbreviation: "Betclic Masc", Category: "Senior", Season: "2025/2026"},
		{ID: "10906", Name: "Liga Betclic Feminina", Abbreviation: "Betclic Fem", Category: "Senior", Season: "2025/2026"},
		{ID: "10903", Name: "Proliga", Abbreviation: "Proliga", Category: "Senior", Season: "2025/2026"},
		{ID: "10904", Name: "1a Divisao Masculina", Abbreviation: "1a Div Masc", Category: "Senior", Season: "2025/2026"},
		{ID: "10907", Name: "1a Divisao Feminina", Abbreviation: "1a Div Fem", Category: "Senior", Season: "2025/2026"},
	}, nil
}

func (f *FPBAPI) GetAthlete(id string) (*scraper.AthleteData, error) {
	key := cache.CacheKey("athlete", id)
	if raw, ok := f.cache.Get(key); ok {
		var a scraper.AthleteData
		if err := json.Unmarshal(raw, &a); err == nil { return &a, nil }
	}
	u := fmt.Sprintf("%s/atletas/%s/", fpbBase, url.PathEscape(id))
	body, err := f.http.Get(u)
	if err != nil { return nil, err }
	a := scraper.ScrapeAthlete(string(body))
	raw2, _ := json.Marshal(a)
	f.cache.Set(key, raw2, cache.TTLHistorical)
	return a, nil
}

func (f *FPBAPI) GetGamesByCompetition(compID, page string) ([]models.Game, error) { return nil, fmt.Errorf("not implemented") }
func (f *FPBAPI) GetTeam(id string) (*scraper.TeamDetail, error)                  { return nil, fmt.Errorf("not implemented") }
func (f *FPBAPI) GetClubTeams(clubID string) ([]models.Team, error)               { return nil, fmt.Errorf("not implemented") }

func (f *FPBAPI) GetTugaBasketStandings(competitionID string) ([]scraper.TugaBasketStanding, error) {
	key := cache.CacheKey("tugabasket", competitionID)
	if raw, ok := f.cache.Get(key); ok {
		var s []scraper.TugaBasketStanding
		if err := json.Unmarshal(raw, &s); err == nil { return s, nil }
	}
	u := fmt.Sprintf("https://resultados.tugabasket.com/getCompetitionDetails?competitionId=%s", competitionID)
	body, err := f.http.Get(u)
	if err != nil { return nil, err }
	standings := scraper.ScrapeTugaBasketStandings(string(body))
	raw2, _ := json.Marshal(standings)
	f.cache.Set(key, raw2, cache.TTLStandings)
	return standings, nil
}
