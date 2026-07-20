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
	"2016/2017", "2017/2018", "2018/2019", "2019/2020",
	"2020/2021", "2021/2022", "2022/2023", "2023/2024", "2024/2025", "2025/2026",
}

// ImportGamesFromSupabase fetches all historical games from Supabase and inserts into SQLite.
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
		n, err := s.importSeasonSimple(table, season)
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

type gameRow struct {
	Slug          string `json:"slug"`
	Data          string `json:"data"`
	Hora          string `json:"hora"`
	EquipaCasa    string `json:"equipa_casa"`
	EquipaFora    string `json:"equipa_fora"`
	ResultadoCasa *int   `json:"resultado_casa"`
	ResultadoFora *int   `json:"resultado_fora"`
	Competicao    string `json:"competicao"`
	Escalao       string `json:"escalao"`
	Local         string `json:"local"`
	Status        string `json:"status"`
	LogotipoCasa  string `json:"logotipo_casa"`
	LogotipoFora  string `json:"logotipo_fora"`
}

func (s *Store) importSeasonSimple(table, season string) (int, error) {
	total := 0

	// Prepare insert statement once
	stmt, err := s.db.Prepare(`INSERT OR REPLACE INTO games (id, season, data, hora, equipa_casa, equipa_fora, resultado_casa, resultado_fora, competicao, escalao, local, status, logo_casa, logo_fora) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil { return 0, err }
	defer stmt.Close()

	offset := 0
	for {
		url := fmt.Sprintf("%s/%s?select=slug,data,hora,equipa_casa,equipa_fora,resultado_casa,resultado_fora,competicao,escalao,local,status,logotipo_casa,logotipo_fora&order=data.asc&limit=1000&offset=%d", supabaseURL, table, offset)

		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("apikey", supabaseAnonKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil { return total, err }

		var games []gameRow
		if err := json.NewDecoder(resp.Body).Decode(&games); err != nil {
			resp.Body.Close()
			return total, fmt.Errorf("decode: %w", err)
		}
		resp.Body.Close()

		if len(games) == 0 { break }

		for _, g := range games {
			id := g.Slug
			if id == "" { continue }
			loc := g.Local; if loc == "" { loc = "-" }
			lc := g.LogotipoCasa; if lc == "" { lc = "-" }
			lf := g.LogotipoFora; if lf == "" { lf = "-" }
			st := g.Status; if st == "" { st = "FINALIZADO" }

			if _, err := stmt.Exec(id, season, g.Data, g.Hora, g.EquipaCasa, g.EquipaFora, g.ResultadoCasa, g.ResultadoFora, g.Competicao, g.Escalao, loc, st, lc, lf); err != nil {
				log.Printf("[import] insert error: %v", err)
				return total, err
			}
			total++
		}

		// Verify immediately after batch
		s.db.Exec("PRAGMA wal_checkpoint(FULL)")
		var verifyCount int
		s.db.QueryRow("SELECT COUNT(*) FROM games WHERE season = ?", season).Scan(&verifyCount)
		log.Printf("[import] %s page at offset %d: inserted %d, DB has %d for this season", season, offset, len(games), verifyCount)

		if len(games) < 1000 { break }
		offset += 1000
	}

	return total, nil
}
