package fpbapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/PuerkitoBio/goquery"
	"github.com/mefrraz/bounce/internal/cache"
	"github.com/mefrraz/bounce/internal/httpclient"
	"github.com/mefrraz/bounce/internal/models"
	"github.com/mefrraz/bounce/internal/scraper"
)

const fpbBase = "https://www.fpb.pt"

type FPBAPI struct {
	http  *httpclient.Client
	cache *cache.Store
}

func New(c *httpclient.Client, s *cache.Store) *FPBAPI { return &FPBAPI{http: c, cache: s} }

func (f *FPBAPI) GetGame(internalID string) (*models.GameDetail, error) {
	key := cache.CacheKey("game", internalID)
	if raw, ok := f.cache.Get(key); ok {
		var g models.GameDetail
		if err := json.Unmarshal(raw, &g); err == nil { return &g, nil }
	}
	body, err := f.http.Get(fmt.Sprintf("%s/ficha-de-jogo?internalID=%s", fpbBase, url.PathEscape(internalID)))
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
		var c []models.Game
		if err := json.Unmarshal(raw, &c); err == nil { return c, nil }
	}
	parts := strings.Split(season, "/")
	if len(parts) != 2 { return nil, fmt.Errorf("invalid season: %s", season) }
	ys, ye := parts[0], parts[1]

	p := url.Values{}
	p.Set("action", "get_more_days")
	p.Set("epoca", season); p.Set("clube", clubID)
	p.Set("period[time_option]", "fromInit")
	p.Set("period[from_date]", ys+"/09/01")
	p.Set("period[to_date]", ye+"/07/31")
	body, err := f.http.Get(fpbBase + "/wp-admin/admin-ajax.php?" + p.Encode())
	if err != nil { return nil, err }
	var ar struct{ Result interface{}; Hasmore bool }
	if err := json.Unmarshal(body, &ar); err != nil { return nil, err }
	var h strings.Builder
	switch v := ar.Result.(type) {
	case string: h.WriteString(v)
	case []interface{}:
		for _, item := range v { if s, ok := item.(string); ok { h.WriteString(s) } }
	}
	all := scraper.ScrapeGames(h.String(), "FINALIZADO")
	log.Printf("[games] %d for club %s season %s", len(all), clubID, season)

	// Enrich scores via get_game_layer POST (parallel, no rate limit)
	scored := int32(0)
	sem := make(chan struct{}, 5) // 2 concurrent requests
	var wg sync.WaitGroup
	for i := range all {
		if all[i].HomeScore != nil { continue }
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			gs := f.fetchScore(all[idx].ID)
			if gs != nil {
				all[idx].HomeScore = gs.HomeScore
				all[idx].AwayScore = gs.AwayScore
				all[idx].Status = "FINALIZADO"
				atomic.AddInt32(&scored, 1)
			}
		}(i)
	}
	wg.Wait()
	log.Printf("[scores] %d/%d enriched for club %s season %s", scored, len(all), clubID, season)

	raw2, _ := json.Marshal(all)
	f.cache.Set(key, raw2, cache.TTLToday)
	return all, nil
}

type gameScore struct{ HomeScore, AwayScore *int }

func (f *FPBAPI) fetchScore(internalID string) *gameScore {
	body := "action=get_game_layer&matchId=" + internalID
	resp, err := f.http.PostFast(fpbBase+"/wp-admin/admin-ajax.php", body)
	if err != nil { return nil }
	var layer struct{ Header string `json:"header"` }
	if err := json.Unmarshal(resp, &layer); err != nil || layer.Header == "" { return nil }
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(layer.Header))
	if err != nil { return nil }
	spans := doc.Find(".points span")
	if spans.Length() < 2 { return nil }
	h := atoi2(strings.TrimSpace(spans.Eq(0).Text()))
	a := atoi2(strings.TrimSpace(spans.Eq(1).Text()))
	if h == 0 && a == 0 { return nil }
	return &gameScore{HomeScore: &h, AwayScore: &a}
}

func atoi2(s string) int {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" { return 0 }
	v := 0; for _, c := range s { v = v*10 + int(c-'0') }; return v
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
		comps = []models.Competition{{ID: "10902", Name: "Liga Betclic"}, {ID: "10903", Name: "Proliga"}, {ID: "10904", Name: "1a Divisao"}}
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
	if td == nil { return nil, fmt.Errorf("parse team %s failed", id) }
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

func (f *FPBAPI) GetStandings(compID string) ([]models.Standing, error) {
	key := cache.CacheKey("standings", compID)
	if raw, ok := f.cache.Get(key); ok {
		var s []models.Standing
		if err := json.Unmarshal(raw, &s); err == nil { return s, nil }
	}
	html, err := f.http.Get(fmt.Sprintf("%s/classificacao/%s", fpbBase, compID))
	faseID := "30969"
	if err == nil {
		for _, fs := range scraper.ExtractFaseIDs(string(html)) { faseID = fs.ID; break }
	}
	body, err := f.http.Get(fmt.Sprintf("%s/wp-admin/admin-ajax.php?action=get_more_fase_regular&competicao%%5B%%5D=%s&fase=%s", fpbBase, compID, faseID))
	if err != nil { return nil, err }
	var ar struct{ Result struct{ Body string `json:"body"` } `json:"result"` }
	var s []models.Standing
	if json.Unmarshal(body, &ar) == nil && ar.Result.Body != "" {
		s = scraper.ScrapeStandings(ar.Result.Body)
	}
	if len(s) == 0 { s = scraper.ScrapeStandings(string(body)) }
	raw2, _ := json.Marshal(s)
	f.cache.Set(key, raw2, cache.TTLStandings)
	return s, nil
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
