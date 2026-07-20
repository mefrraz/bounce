package fpbapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	"github.com/mefrraz/bounce/internal/cache"
	"github.com/mefrraz/bounce/internal/clubs"
	"github.com/mefrraz/bounce/internal/httpclient"
	"github.com/mefrraz/bounce/internal/metrics"
	"github.com/mefrraz/bounce/internal/models"
	"github.com/mefrraz/bounce/internal/scraper"
)

const fpbBase = "https://www.fpb.pt"

type FPBAPI struct {
	http  *httpclient.Client
	cache *cache.Store
}

func New(c *httpclient.Client, s *cache.Store) *FPBAPI { return &FPBAPI{http: c, cache: s} }
func (f *FPBAPI) Cache() *cache.Store { return f.cache }

// normalizeGames replaces raw FPB team names/logos with canonical club data where possible.
// Also auto-detects new clubs for team names not matching any known club.
func normalizeGames(games []models.Game) {
	for i := range games {
		g := &games[i]
		homeName, homeLogo := clubs.NormalizeTeam(g.HomeTeam, g.HomeLogo)
		awayName, awayLogo := clubs.NormalizeTeam(g.AwayTeam, g.AwayLogo)
		// If team name wasn't normalized, try to add as new club
		if homeName == g.HomeTeam && g.HomeTeam != "" && g.HomeLogo != "" {
			clubs.MaybeAddPending(g.HomeTeam, g.HomeLogo)
		}
		if awayName == g.AwayTeam && g.AwayTeam != "" && g.AwayLogo != "" {
			clubs.MaybeAddPending(g.AwayTeam, g.AwayLogo)
		}
		g.HomeTeam = homeName
		g.HomeLogo = homeLogo
		g.AwayTeam = awayName
		g.AwayLogo = awayLogo
	}
}

func (f *FPBAPI) GetGame(internalID string) (*models.GameDetail, error) {
	key := cache.CacheKey("game", internalID)
	if raw, ok := f.cache.Get(key); ok { metrics.IncCacheHit()
		var g models.GameDetail
		if err := json.Unmarshal(raw, &g); err == nil { return &g, nil }
	}
	metrics.IncCacheMiss()
	metrics.IncFPBRequest()
	body, err := f.http.Get(fmt.Sprintf("%s/ficha-de-jogo?internalID=%s", fpbBase, url.PathEscape(internalID)))
	if err != nil { return nil, err }
	detail, err := scraper.ScrapeGameDetail(string(body))
	if err != nil { return nil, fmt.Errorf("parse game %s: %w", internalID, err) }
	detail.ID = internalID

	// Normalize team names using clubs data
	homeName, homeLogo := clubs.NormalizeTeam(detail.HomeTeam, detail.HomeLogo)
	awayName, awayLogo := clubs.NormalizeTeam(detail.AwayTeam, detail.AwayLogo)
	detail.HomeTeam = homeName
	detail.HomeLogo = homeLogo
	detail.AwayTeam = awayName
	detail.AwayLogo = awayLogo

	// Chain invalidation: status change → invalidate club calendars
	if oldRaw, ok := f.cache.Get(key); ok {
		var old models.GameDetail
		if json.Unmarshal(oldRaw, &old) == nil && old.Status != "" && old.Status != detail.Status {
			log.Printf("[invalidate] game %s: %s → %s", internalID, old.Status, detail.Status)
			f.cache.Invalidate("games:club:")
		}
	}

	ttl := cache.TTLForGame(detail.Date, detail.Status)
	raw2, _ := json.Marshal(detail)
	f.cache.Set(key, raw2, ttl)
	return detail, nil
}

func (f *FPBAPI) GetGamesByClub(clubID, season, category, gender string) ([]models.Game, error) {
	key := cache.CacheKey("games", "club", clubID, season, category, gender)
	if raw, ok := f.cache.Get(key); ok { metrics.IncCacheHit()
		var c []models.Game
		if err := json.Unmarshal(raw, &c); err == nil { return c, nil }
	}
	metrics.IncCacheMiss()
	p := url.Values{}
	p.Set("action", "get_results")
	p.Set("epoca", season)
	p.Set("clube", clubID)
	if category != "" { p.Set("escalao", category) }
	if gender != "" { p.Set("genero", gender) }
	metrics.IncFPBRequest()
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
	normalizeGames(all)

	// Persist to SQLite games table
	for _, g := range all {
		if g.ID == "" || g.Date == "" { continue }
		f.cache.UpsertGame(g.ID, season, g.Date, g.Time,
			g.HomeTeam, g.AwayTeam, g.Competition, g.Category, g.Venue, g.Status, g.HomeLogo, g.AwayLogo,
			g.HomeScore, g.AwayScore)
	}

	hasToday := false
	for _, g := range all {
		if cache.IsToday(g.Date) { hasToday = true; break }
	}
	ttl := cache.TTLForCalendar(hasToday, cache.IsOffSeason())
	raw2, _ := json.Marshal(all)
	f.cache.Set(key, raw2, ttl)
	return all, nil
}

func (f *FPBAPI) GetCompetitions(category, gender string) ([]models.Competition, error) {
	if category == "" { category = "Senior" }
	if gender == "" { gender = "masculino" }
	key := cache.CacheKey("competitions", category, gender)
	if raw, ok := f.cache.Get(key); ok { metrics.IncCacheHit()
		var c []models.Competition
		if err := json.Unmarshal(raw, &c); err == nil { return c, nil }
	}
	metrics.IncCacheMiss()
	metrics.IncFPBRequest()
	body, err := f.http.Post(fpbBase+"/wp-admin/admin-ajax.php",
		"action=get_competicoes&epoca="+cache.CurrentSeason()+"&escalao="+url.QueryEscape(category)+"&genero="+url.QueryEscape(gender)+"&radio=true")
	var comps []models.Competition
	if err == nil {
		re := regexp.MustCompile(`data-id="(\d+)"[^>]*>\s*<span[^>]*>([^<]+)</span>`)
		for _, m := range re.FindAllStringSubmatch(string(body), -1) {
			comps = append(comps, models.Competition{ID: m[1], Name: strings.TrimSpace(m[2])})
		}
		// Fallback: try without span
		if len(comps) == 0 {
			re2 := regexp.MustCompile(`data-id="(\d+)"[^>]*>\s*([^<]+)`)
			for _, m := range re2.FindAllStringSubmatch(string(body), -1) {
				comps = append(comps, models.Competition{ID: m[1], Name: strings.TrimSpace(m[2])})
			}
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
	if raw, ok := f.cache.Get(key); ok { metrics.IncCacheHit()
		var a scraper.AthleteData
		if err := json.Unmarshal(raw, &a); err == nil { return &a, nil }
	}
	metrics.IncCacheMiss()
	metrics.IncFPBRequest()
	body, err := f.http.Get(fmt.Sprintf("%s/atletas/%s/", fpbBase, url.PathEscape(id)))
	if err != nil { return nil, err }
	a := scraper.ScrapeAthlete(string(body))
	raw2, _ := json.Marshal(a)
	f.cache.Set(key, raw2, 1440)
	return a, nil
}

func (f *FPBAPI) GetTeam(id string) (*scraper.TeamDetail, error) {
	key := cache.CacheKey("team", id)
	if raw, ok := f.cache.Get(key); ok { metrics.IncCacheHit()
		var td scraper.TeamDetail
		if err := json.Unmarshal(raw, &td); err == nil { return &td, nil }
	}
	metrics.IncCacheMiss()
	metrics.IncFPBRequest()
	body, err := f.http.Get(fmt.Sprintf("%s/equipa/%s/", fpbBase, url.PathEscape(id)))
	if err != nil { return nil, err }
	td := scraper.ScrapeTeamDetail(string(body))
	raw2, _ := json.Marshal(td)
	f.cache.Set(key, raw2, 1440)
	return td, nil
}

func (f *FPBAPI) GetClubTeams(clubID string) ([]models.Team, error) {
	key := cache.CacheKey("clubteams", clubID)
	if raw, ok := f.cache.Get(key); ok { metrics.IncCacheHit()
		var t []models.Team
		if err := json.Unmarshal(raw, &t); err == nil { return t, nil }
	}
	metrics.IncCacheMiss()
	metrics.IncFPBRequest()
	body, err := f.http.Get(fmt.Sprintf("%s/wp-admin/admin-ajax.php?action=get_equipas&idClube=%s&epoca=2025/2026epoca="+cache.CurrentSeason()+"", fpbBase, clubID))
	if err != nil { return nil, err }
	teams := scraper.ScrapeClubTeams(string(body))
	raw2, _ := json.Marshal(teams)
	f.cache.Set(key, raw2, 1440)
	return teams, nil
}

func (f *FPBAPI) GetStandings(compID string) ([]models.Standing, error) {
	key := cache.CacheKey("standings", compID)
	if raw, ok := f.cache.Get(key); ok { metrics.IncCacheHit()
		var s []models.Standing
		if err := json.Unmarshal(raw, &s); err == nil { return s, nil }
	}
	html, _ := f.http.Get(fmt.Sprintf("%s/classificacao/%s", fpbBase, compID))
	faseID := "30969"
	if html != nil {
		for _, fs := range scraper.ExtractFaseIDs(string(html)) { faseID = fs.ID; break }
	}
	metrics.IncCacheMiss()
	metrics.IncFPBRequest()
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
	if raw, ok := f.cache.Get(key); ok { metrics.IncCacheHit()
		var s []scraper.TugaBasketStanding
		if err := json.Unmarshal(raw, &s); err == nil { return s, nil }
	}
	metrics.IncCacheMiss()
	metrics.IncFPBRequest()
	body, err := f.http.Get(fmt.Sprintf("https://resultados.tugabasket.com/getCompetitionDetails?competitionId=%s", competitionID))
	if err != nil { return nil, err }
	s := scraper.ScrapeTugaBasketStandings(string(body))
	raw2, _ := json.Marshal(s)
	f.cache.Set(key, raw2, cache.TTLStandings)
	return s, nil
}

func (f *FPBAPI) GetTugaBasketPlayers(competitionID string) ([]scraper.TBPlayerStat, error) {
	key := cache.CacheKey("tugabasket_players", competitionID)
	if raw, ok := f.cache.Get(key); ok { metrics.IncCacheHit()
		var p []scraper.TBPlayerStat
		if err := json.Unmarshal(raw, &p); err == nil { return p, nil }
	}
	metrics.IncCacheMiss()
	metrics.IncFPBRequest()
	body, err := f.http.Get(fmt.Sprintf("https://resultados.tugabasket.com/stats/players?competitionId=%s", competitionID))
	if err != nil { return nil, err }
	p := scraper.ScrapeTugaBasketPlayers(string(body))
	raw2, _ := json.Marshal(p)
	f.cache.Set(key, raw2, cache.TTLStandings)
	return p, nil
}

func (f *FPBAPI) GetTugaBasketTeams(competitionID string) ([]scraper.TBTeamStat, error) {
	key := cache.CacheKey("tugabasket_teams", competitionID)
	if raw, ok := f.cache.Get(key); ok { metrics.IncCacheHit()
		var t []scraper.TBTeamStat
		if err := json.Unmarshal(raw, &t); err == nil { return t, nil }
	}
	metrics.IncCacheMiss()
	metrics.IncFPBRequest()
	body, err := f.http.Get(fmt.Sprintf("https://resultados.tugabasket.com/stats/teams?competitionId=%s", competitionID))
	if err != nil { return nil, err }
	t := scraper.ScrapeTugaBasketTeams(string(body))
	raw2, _ := json.Marshal(t)
	f.cache.Set(key, raw2, cache.TTLStandings)
	return t, nil
}

func (f *FPBAPI) GetGamesByCompetition(compID, season string) ([]models.Game, error) {
	key := cache.CacheKey("games", "comp", compID, season)
	if raw, ok := f.cache.Get(key); ok { metrics.IncCacheHit()
		var g []models.Game
		if err := json.Unmarshal(raw, &g); err == nil { return g, nil }
	}
	metrics.IncCacheMiss()
	p := url.Values{}
	p.Set("action", "get_results")
	p.Set("epoca", season)
	p.Set("competicao[]", compID)
	metrics.IncFPBRequest()
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
	g := scraper.ScrapeGames(h.String(), "FINALIZADO")
	normalizeGames(g)
	raw2, _ := json.Marshal(g)
	f.cache.Set(key, raw2, cache.TTLStandings)
	return g, nil
}


type CompMVP struct {
	Category string `json:"categoria"`
	Player   string `json:"jogador"`
	Team     string `json:"equipa"`
	Value    string `json:"valor"`
}

func (f *FPBAPI) GetCompetitionMVP(compID string) ([]CompMVP, error) {
	key := cache.CacheKey("compmvp", compID)
	if raw, ok := f.cache.Get(key); ok {
		var m []CompMVP
		if err := json.Unmarshal(raw, &m); err == nil { return m, nil }
	}
	body, err := f.http.Get(fmt.Sprintf("https://sav2.fpb.pt/api/mvp/prova/%s", compID))
	if err != nil {
		if raw, ok := f.cache.GetStale(key); ok {
			var m []CompMVP
			if json.Unmarshal(raw, &m) == nil { return m, nil }
		}
		return nil, err
	}
	var raw []struct {
		Category string `json:"category"`
		Player   string `json:"player"`
		Team     string `json:"team"`
		Value    string `json:"value"`
	}
	if err := json.Unmarshal(body, &raw); err != nil { return nil, err }
	var result []CompMVP
	for _, r := range raw {
		result = append(result, CompMVP{Category: r.Category, Player: r.Player, Team: r.Team, Value: r.Value})
	}
	raw2, _ := json.Marshal(result)
	f.cache.Set(key, raw2, 60)
	return result, nil
}
