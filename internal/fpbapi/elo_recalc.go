package fpbapi

import (
	"fmt"
	"log"
	"strings"

	"github.com/mefrraz/bounce/internal/clubs"
	"github.com/mefrraz/bounce/internal/elo"
)

// RecalculateELO computes ELO for a season from the local games table and stores results.
func (f *FPBAPI) RecalculateELO(season string) error {
	store := elo.NewStore(f.cache.DB())

	// 1. Fetch games from local SQLite
	gameRows, err := f.cache.GetGamesBySeason(season)
	if err != nil {
		return fmt.Errorf("get games for %s: %w", season, err)
	}
	if len(gameRows) == 0 {
		return fmt.Errorf("no games found for %s", season)
	}

	// 2. Compute ELO
	eloGames := make([]elo.Game, len(gameRows))
	for i, g := range gameRows {
		eloGames[i] = elo.Game{
			HomeTeam:     g.HomeTeam,
			AwayTeam:     g.AwayTeam,
			HomeScore:    g.HomeScore,
			AwayScore:    g.AwayScore,
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

	log.Printf("[elo] %s: %d games → %d clubs rated", season, len(gameRows), len(rows))
	return nil
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
	for clubName, id := range clubMap {
		if strings.Contains(clubName, n) || strings.Contains(n, clubName) {
			return id
		}
	}
	return 0
}
