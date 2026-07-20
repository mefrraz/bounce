package cache

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

const supabaseAnonKey = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6InFkem13Z2FoZW5jaW5vdWN2b29wIiwicm9sZSI6ImFub24iLCJpYXQiOjE3Njk5NTQ2NTEsImV4cCI6MjA4NTUzMDY1MX0.HNcyu7zHA6oxBNh0T7HX-6Ui-8g2fBE5gFP4xtkpPJ4"
const supabaseURL = "https://qdzmwgahencinoucvoop.supabase.co/rest/v1"

var seasonsToImport = []string{
	"2003/2004", "2004/2005", "2005/2006", "2006/2007", "2007/2008",
	"2008/2009", "2009/2010", "2010/2011", "2011/2012", "2012/2013",
	"2013/2014", "2014/2015", "2015/2016", "2016/2017", "2017/2018",
	"2018/2019", "2019/2020", "2020/2021", "2021/2022", "2022/2023",
	"2023/2024", "2024/2025", "2025/2026",
}

// ImportGamesFromSupabase fetches all historical games from Supabase and inserts into SQLite.
// Only runs once (detected by empty games table). Returns seasons that got data.
func (s *Store) ImportGamesFromSupabase() []string {
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM games").Scan(&count)
	if count > 0 {
		log.Printf("[import] games table already has %d rows, skipping", count)
		return nil
	}

	log.Printf("[import] starting Supabase → SQLite migration for %d seasons", len(seasonsToImport))
	start := time.Now()
	total := 0
	var imported []string

	for _, season := range seasonsToImport {
		table := "games_" + strings.ReplaceAll(season, "/", "_")
		n, err := s.importSeason(table, season)
		if err != nil {
			log.Printf("[import] %s: error: %v", season, err)
			continue
		}
		total += n
		if n > 0 { imported = append(imported, season) }
		log.Printf("[import] %s: %d games", season, n)
	}

	log.Printf("[import] done: %d games in %v", total, time.Since(start).Round(time.Second))
	return imported
}

func (s *Store) importSeason(table, season string) (int, error) {
	total := 0
	offset := 0
	limit := 1000

	for {
		url := fmt.Sprintf("%s/%s?select=*&order=data.asc&limit=%d&offset=%d", supabaseURL, table, limit, offset)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil { return total, err }
		req.Header.Set("apikey", supabaseAnonKey)
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil { return total, err }
		defer resp.Body.Close()

		var raw json.RawMessage
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			return total, err
		}

		var games []struct {
			Slug           string `json:"slug"`
			Data           string `json:"data"`
			Hora           string `json:"hora"`
			EquipaCasa     string `json:"equipa_casa"`
			EquipaFora     string `json:"equipa_fora"`
			ResultadoCasa  *int   `json:"resultado_casa"`
			ResultadoFora  *int   `json:"resultado_fora"`
			Competicao     string `json:"competicao"`
			Escalao        string `json:"escalao"`
			Local          string `json:"local"`
			Status         string `json:"status"`
			LogotipoCasa   string `json:"logotipo_casa"`
			LogotipoFora   string `json:"logotipo_fora"`
		}

		if err := json.Unmarshal(raw, &games); err != nil {
			var wrapper struct{ Data json.RawMessage `json:"data"` }
			if err2 := json.Unmarshal(raw, &wrapper); err2 != nil || len(wrapper.Data) == 0 {
				// No more data or error — stop paginating
				if total == 0 {
					return 0, fmt.Errorf("unexpected format: %s", string(raw[:min(len(raw), 100)]))
				}
				break
			}
			if err := json.Unmarshal(wrapper.Data, &games); err != nil {
				if total == 0 { return 0, err }
				break
			}
		}

		if len(games) == 0 { break }

		n, err := s.insertGames(games, season)
		if err != nil { return total, err }
		total += n

		if len(games) < limit { break }
		offset += limit
	}
	return total, nil
}

func (s *Store) insertGames(games []struct {
	Slug           string `json:"slug"`
	Data           string `json:"data"`
	Hora           string `json:"hora"`
	EquipaCasa     string `json:"equipa_casa"`
	EquipaFora     string `json:"equipa_fora"`
	ResultadoCasa  *int   `json:"resultado_casa"`
	ResultadoFora  *int   `json:"resultado_fora"`
	Competicao     string `json:"competicao"`
	Escalao        string `json:"escalao"`
	Local          string `json:"local"`
	Status         string `json:"status"`
	LogotipoCasa   string `json:"logotipo_casa"`
	LogotipoFora   string `json:"logotipo_fora"`
}, season string) (int, error) {
	tx, err := s.db.Begin()
	if err != nil { return 0, err }
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO games (id, season, data, hora, equipa_casa, equipa_fora, resultado_casa, resultado_fora, competicao, escalao, local, status, logo_casa, logo_fora) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil { return 0, err }
	defer stmt.Close()

	for _, g := range games {
		loc := g.Local
		if loc == "" { loc = "-" }
		logoC := g.LogotipoCasa
		if logoC == "" { logoC = "-" }
		logoF := g.LogotipoFora
		if logoF == "" { logoF = "-" }
		status := g.Status
		if status == "" { status = "FINALIZADO" }
		// Use Supabase slug as primary key (unique per game)
		id := g.Slug
		if id == "" {
			id = g.Data + "-" + strings.ToLower(g.EquipaCasa) + "-" + strings.ToLower(g.EquipaFora)
			id = strings.ReplaceAll(id, " ", "-")
		}

		if _, err := stmt.Exec(id, season, g.Data, g.Hora, g.EquipaCasa, g.EquipaFora, g.ResultadoCasa, g.ResultadoFora, g.Competicao, g.Escalao, loc, status, logoC, logoF); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil { return 0, err }
	return len(games), nil
}
