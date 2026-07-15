package fpbapi

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
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

func (f *FPBAPI) GetGame(internalID string) (*models.GameDetail, error) {
	key := cache.CacheKey("game", internalID)
	if raw, ok := f.cache.Get(key); ok {
		var g models.GameDetail
		if err := json.Unmarshal(raw, &g); err == nil { return &g, nil }
	}
	u := fmt.Sprintf("%s/ficha-de-jogo?internalID=%s", fpbBase, url.PathEscape(internalID))
	body, err := f.http.Get(u)
	if err != nil { return nil, err }
	detail, _ := scraper.ScrapeGameDetail(string(body))
	detail.ID = internalID
	raw2, _ := json.Marshal(detail)
	f.cache.Set(key, raw2, cache.TTLRecent)
	return detail, nil
}

func (f *FPBAPI) GetGamesByClub(clubID, season, category, gender string) ([]models.Game, error) {
	key := cache.CacheKey("games", "club", clubID, season)
	if raw, ok := f.cache.Get(key); ok {
		var cached []models.Game
		if err := json.Unmarshal(raw, &cached); err == nil { return cached, nil }
	}
	parts := strings.Split(season, "/")
	if len(parts) != 2 { return nil, fmt.Errorf("invalid season: %s", season) }
	yearStart, yearEnd := parts[0], parts[1]

	categories := []struct{ e, g string }{
		{"Sénior", "masculino"}, {"Sénior", "feminino"},
		{"Sub 20", "masculino"}, {"Sub 18", "masculino"}, {"Sub-19", "feminino"},
		{"Sub 16", "masculino"}, {"Sub-16", "feminino"},
		{"Sub 14", "masculino"}, {"Sub-14", "feminino"},
	}
	var all []models.Game
	seen := map[string]bool{}
	for _, cat := range categories {
		p := url.Values{}
		p.Set("action", "get_more_days")
		p.Set("epoca", season); p.Set("escalao", cat.e); p.Set("genero", cat.g)
		p.Set("clube", clubID)
		p.Set("period[time_option]", "fromInit")
		p.Set("period[from_date]", yearStart+"/09/01")
		p.Set("period[to_date]", yearEnd+"/06/30")
		body, err := f.http.Get(fpbBase + "/wp-admin/admin-ajax.php?" + p.Encode())
		if err != nil { continue }
		var ar struct{ Result interface{}; Hasmore bool }
		if err := json.Unmarshal(body, &ar); err != nil { continue }
		var h strings.Builder
		switch v := ar.Result.(type) {
		case string: h.WriteString(v)
		case []interface{}:
			for _, item := range v { if s, ok := item.(string); ok { h.WriteString(s) } }
		}
		games := scraper.ScrapeGames(h.String(), "FINALIZADO")
		for _, g := range games {
			if !seen[g.ID] { seen[g.ID] = true; g.Category = cat.e + " " + cat.g; g.Escalao = cat.e; all = append(all, g) }
		}
	}
	raw2, _ := json.Marshal(all)
	f.cache.Set(key, raw2, cache.TTLToday)
	return all, nil
}

func (f *FPBAPI) GetCompetitions() ([]models.Competition, error) {
	key := cache.CacheKey("competitions")
	if raw, ok := f.cache.Get(key); ok {
		var c []models.Competition
		if err := json.Unmarshal(raw, &c); err == nil { return c, nil }
	}
	body, err := f.http.Post(fpbBase+"/wp-admin/admin-ajax.php", "action=get_competicoes&epoca=2025/2026&escalao=Senior&genero=masculino&radio=true")
	var comps []models.Competition
	if err == nil {
		re := regexp.MustCompile(`data-id="(\d+)"[^>]*>\s*(?:<[^>]+>)*\s*([^<]+)\s*<`)
		for _, m := range re.FindAllStringSubmatch(string(body), -1) {
			comps = append(comps, models.Competition{ID: m[1], Name: strings.TrimSpace(m[2])})
		}
	}
	if len(comps) == 0 {
		comps = []models.Competition{
			{ID: "10902", Name: "Liga Betclic Masculina"},
			{ID: "10903", Name: "Proliga"},
			{ID: "10904", Name: "1a Divisao Masculina"},
		}
	}
	raw2, _ := json.Marshal(comps)
	f.cache.Set(key, raw2, 1440)
	return comps, nil
}

func (f *FPBAPI) GetAthlete(id string) (*scraper.AthleteData, error) {
	key := cache.CacheKey("athlete", id)
	if raw, ok := f.cache.Get(key); ok {
		var a scraper.AthleteData
		if err := json.Unmarshal(raw, &a); err == nil { return &a, nil }
	}
	body, err := f.http.Get(fmt.Sprintf("%s/atletas/%s/", fpbBase, url.PathEscape(id)))
	if err != nil { return nil, err }
	a := scraper.ScrapeAthlete(string(body))
	raw2, _ := json.Marshal(a)
	f.cache.Set(key, raw2, cache.TTLHistorical)
	return a, nil
}

func (f *FPBAPI) GetTeam(id string) (*scraper.TeamDetail, error) {
	key := cache.CacheKey("team", id)
	if raw, ok := f.cache.Get(key); ok {
		var td scraper.TeamDetail
		if err := json.Unmarshal(raw, &td); err == nil { return &td, nil }
	}
	body, err := f.http.Get(fmt.Sprintf("%s/equipa/%s/", fpbBase, url.PathEscape(id)))
	if err != nil { return nil, err }
	td := scraper.ScrapeTeamDetail(string(body))
	raw2, _ := json.Marshal(td)
	f.cache.Set(key, raw2, cache.TTLHistorical)
	return td, nil
}

func (f *FPBAPI) GetClubTeams(clubID string) ([]models.Team, error) {
	key := cache.CacheKey("clubteams", clubID)
	if raw, ok := f.cache.Get(key); ok {
		var t []models.Team
		if err := json.Unmarshal(raw, &t); err == nil { return t, nil }
	}
	body, err := f.http.Get(fmt.Sprintf("%s/wp-admin/admin-ajax.php?action=get_equipas&idClube=%s&epoca=2025/2026", fpbBase, clubID))
	if err != nil { return nil, err }
	teams := scraper.ScrapeClubTeams(string(body))
	raw2, _ := json.Marshal(teams)
	f.cache.Set(key, raw2, cache.TTLHistorical)
	return teams, nil
}

func (f *FPBAPI) GetTugaBasketStandings(competitionID string) ([]scraper.TugaBasketStanding, error) {
	key := cache.CacheKey("tugabasket", competitionID)
	if raw, ok := f.cache.Get(key); ok {
		var s []scraper.TugaBasketStanding
		if err := json.Unmarshal(raw, &s); err == nil { return s, nil }
	}
	body, err := f.http.Get(fmt.Sprintf("https://resultados.tugabasket.com/getCompetitionDetails?competitionId=%s", competitionID))
	if err != nil { return nil, err }
	s := scraper.ScrapeTugaBasketStandings(string(body))
	raw2, _ := json.Marshal(s)
	f.cache.Set(key, raw2, cache.TTLStandings)
	return s, nil
}

func (f *FPBAPI) GetStandings(compID string) ([]models.Standing, error) {
	key := cache.CacheKey("standings", compID)
	if raw, ok := f.cache.Get(key); ok {
		var s []models.Standing
		if err := json.Unmarshal(raw, &s); err == nil { return s, nil }
	}
	body, err := f.http.Get(fmt.Sprintf("https://sav2.fpb.pt/api/classificacao/%s", compID))
	if err != nil { return nil, err }
	var s []models.Standing
	json.Unmarshal(body, &s)
	raw2, _ := json.Marshal(s)
	f.cache.Set(key, raw2, cache.TTLStandings)
	return s, nil
}

func (f *FPBAPI) GetGamesByCompetition(compID, page string) ([]models.Game, error) { return nil, fmt.Errorf("wip") }
