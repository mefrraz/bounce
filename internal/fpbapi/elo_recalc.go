package fpbapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/mefrraz/bounce/internal/clubs"
	"github.com/mefrraz/bounce/internal/elo"
)

// RecalculateELO fetches all games for a season, computes ELO, and stores results.
func (f *FPBAPI) RecalculateELO(season string) error {
	store := elo.NewStore(f.cache.DB())

	// 1. Fetch all games for this season from local cache/FPB
	games, err := f.fetchAllGamesForSeason(season)
	if err != nil {
		return fmt.Errorf("fetch games for %s: %w", season, err)
	}
	if len(games) == 0 {
		return fmt.Errorf("no games found for %s", season)
	}

	// 2. Compute ELO
	eloGames := make([]elo.Game, len(games))
	for i, g := range games {
		eloGames[i] = elo.Game{
			HomeTeam:    g.HomeTeam,
			AwayTeam:    g.AwayTeam,
			HomeScore:   g.HomeScore,
			AwayScore:   g.AwayScore,
			HomePriority: f.clubPriority(g.HomeTeam),
			AwayPriority: f.clubPriority(g.AwayTeam),
		}
	}

	results := elo.Calculate(eloGames)
	clubMap := f.clubNameMap()

	// 3. Match team names to club IDs and store
	var rows []elo.RatingRow
	for _, r := range results {
		clubID := matchClub(r.Team, clubMap)
		if clubID > 0 {
			rows = append(rows, elo.RatingRow{
				ClubID:      clubID,
				Season:      season,
				EloRating:   r.Rating,
				GamesPlayed: r.GamesPlayed,
			})
		}
	}

	if err := store.BatchUpsert(rows); err != nil {
		return fmt.Errorf("store elo: %w", err)
	}

	log.Printf("[elo] %s: %d games → %d clubs rated", season, len(games), len(rows))
	return nil
}

// Simple game struct for internal use.
type seasonGame struct {
	HomeTeam  string
	AwayTeam  string
	HomeScore int
	AwayScore int
}

// fetchAllGamesForSeason gets all games for a season by fetching from FPB for all known clubs.
// For now, returns empty — full implementation needs Supabase or FPB scrape.
func (f *FPBAPI) fetchAllGamesForSeason(season string) ([]seasonGame, error) {
	// TODO: Phase 2 — fetch all games from Supabase or FPB
	log.Printf("[elo] fetchAllGamesForSeason %s: using Supabase games table", season)
	return f.fetchFromSupabase(season)
}

// fetchFromSupabase reads games from the Supabase games_{season} table.
func (f *FPBAPI) fetchFromSupabase(season string) ([]seasonGame, error) {
	tableName := strings.ReplaceAll("games_"+season, "/", "_")
	url := fmt.Sprintf("https://qdzmwgahencinoucvoop.supabase.co/rest/v1/%s?select=equipa_casa,equipa_fora,resultado_casa,resultado_fora&not.resultado_casa=is.null&not.resultado_fora=is.null&order=data.asc", tableName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil { return nil, err }
	req.Header.Set("apikey", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6InFkem13Z2FoZW5jaW5vdWN2b29wIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NDM5NDg2OTQsImV4cCI6MjA1OTUyNDY5NH0._XWQ0td2LQ5Xb-XbS8xeeI1-L-qSc6uFe7EvZKX_SZY")

	resp, err := http.DefaultClient.Do(req)
	if err != nil { return nil, err }
	defer resp.Body.Close()

	var games []struct {
		EquipaCasa    string `json:"equipa_casa"`
		EquipaFora    string `json:"equipa_fora"`
		ResultadoCasa *int   `json:"resultado_casa"`
		ResultadoFora *int   `json:"resultado_fora"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&games); err != nil {
		return nil, err
	}

	var out []seasonGame
	for _, g := range games {
		hc := 0
		ac := 0
		if g.ResultadoCasa != nil { hc = *g.ResultadoCasa }
		if g.ResultadoFora != nil { ac = *g.ResultadoFora }
		out = append(out, seasonGame{
			HomeTeam:  strings.TrimSpace(g.EquipaCasa),
			AwayTeam:  strings.TrimSpace(g.EquipaFora),
			HomeScore: hc,
			AwayScore: ac,
		})
	}
	return out, nil
}

// clubPriority returns the priority level for a team name, defaulting to 4.
func (f *FPBAPI) clubPriority(name string) int {
	n := strings.ToLower(strings.TrimSpace(name))
	for _, c := range clubs.All() {
		if c.Priority == 0 { continue }
		cn := strings.ToLower(strings.TrimSpace(c.Name))
		cs := strings.ToLower(strings.TrimSpace(c.ShortName))
		if n == cn || n == cs || strings.Contains(cn, n) || strings.Contains(n, cn) {
			return c.Priority
		}
	}
	return 4
}

// clubNameMap returns a map of normalized name → club ID.
func (f *FPBAPI) clubNameMap() map[string]int {
	m := make(map[string]int)
	for _, c := range clubs.All() {
		key := strings.ToLower(strings.TrimSpace(c.Name))
		m[key] = c.ID
		if c.ShortName != "" {
			m[strings.ToLower(strings.TrimSpace(c.ShortName))] = c.ID
		}
	}
	return m
}

// matchClub finds the club ID for a team name using fuzzy matching.
func matchClub(name string, clubMap map[string]int) int {
	if clubMap == nil { return 0 }
	n := strings.ToLower(strings.TrimSpace(name))
	if id, ok := clubMap[n]; ok { return id }
	// Try substring match
	for clubName, id := range clubMap {
		if strings.Contains(clubName, n) || strings.Contains(n, clubName) {
			return id
		}
	}
	return 0
}
