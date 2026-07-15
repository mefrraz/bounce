package fpbapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

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
	p.Set("period[to_date]", ye+"/06/30")
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
	tb := f.fetchTugaScores()
	for i := range all {
		if all[i].HomeScore != nil { continue }
		isoD := scraper.ParseDatePt(all[i].Date)
		for _, t := range tb {
			if matchTeam(all[i].HomeTeam, t.HomeTeam) && matchTeam(all[i].AwayTeam, t.AwayTeam) && (isoD == t.Date || all[i].Date == t.Date || strings.Contains(all[i].Date, t.Date) || strings.Contains(t.Date, isoD)) {
				s := strings.Split(t.Score, ":")
				if len(s) == 2 {
					hs, as := atoi2(s[0]), atoi2(s[1])
					all[i].HomeScore = &hs; all[i].AwayScore = &as; all[i].Status = "FINALIZADO"
				}
				break
			}
		}
	}
	log.Printf("[games] %d for club %s season %s", len(all), clubID, season)
	raw2, _ := json.Marshal(all)
	f.cache.Set(key, raw2, cache.TTLToday)
	return all, nil
}

func (f *FPBAPI) fetchTugaScores() []scraper.TBGameResult {
	var a []scraper.TBGameResult
	for _, cid := range []string{"10902","10903","10904","10906","10907"} {
		b, err := f.http.Get(fmt.Sprintf("https://resultados.tugabasket.com/getCompetitionDetails?competitionId=%s", cid))
		if err != nil { continue }
		a = append(a, scraper.ScrapeTugaBasketGames(string(b))...)
	}
	return a
}

func matchTeam(a, b string) bool {
	a, b = strings.ToUpper(strings.TrimSpace(a)), strings.ToUpper(strings.TrimSpace(b))
	if a == b { return true }
	if len(a) > 5 && len(b) > 5 && (strings.Contains(a, b) || strings.Contains(b, a)) { return true }
	return false
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
