package cache

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

const (
	TTLStandings  = 60
	TTLHistorical = 1440
)

type Store struct {
	db *sql.DB
}

func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS cache_entries (
		key TEXT PRIMARY KEY, value BLOB NOT NULL,
		ttl_min INTEGER NOT NULL,
		created_at INTEGER NOT NULL DEFAULT (unixepoch())
	)`)
	if err != nil { return err }
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS metrics_snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		time INTEGER NOT NULL,
		requests INTEGER NOT NULL DEFAULT 0,
		cache_hits INTEGER NOT NULL DEFAULT 0,
		cache_misses INTEGER NOT NULL DEFAULT 0,
		fpb_requests INTEGER NOT NULL DEFAULT 0,
		rate_limited INTEGER NOT NULL DEFAULT 0,
		goroutines INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil { return err }
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_metrics_time ON metrics_snapshots(time)`)
	if err != nil { return err }
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS elo_history (
		club_id INTEGER NOT NULL,
		season TEXT NOT NULL,
		elo_rating REAL NOT NULL DEFAULT 1500,
		games_played INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL DEFAULT (unixepoch()),
		PRIMARY KEY (club_id, season)
	)`)
	if err != nil { return err }
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_elo_season ON elo_history(season)`)
	if err != nil { return err }
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS games (
		season TEXT NOT NULL DEFAULT '',
		id TEXT NOT NULL DEFAULT '',
		data TEXT NOT NULL DEFAULT '',
		hora TEXT NOT NULL DEFAULT '',
		equipa_casa TEXT NOT NULL DEFAULT '',
		equipa_fora TEXT NOT NULL DEFAULT '',
		resultado_casa INTEGER,
		resultado_fora INTEGER,
		competicao TEXT NOT NULL DEFAULT '',
		escalao TEXT NOT NULL DEFAULT '',
		local TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'AGENDADO',
		logo_casa TEXT NOT NULL DEFAULT '',
		logo_fora TEXT NOT NULL DEFAULT '',
		created_at INTEGER NOT NULL DEFAULT (unixepoch()),
		updated_at INTEGER NOT NULL DEFAULT (unixepoch()),
		PRIMARY KEY (season, id)
	)`)
	if err != nil { return err }
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_games_season ON games(season, data)`)
	if err != nil { return err }
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_games_home ON games(equipa_casa)`)
	if err != nil { return err }
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_games_away ON games(equipa_fora)`)
	return err
}

func (s *Store) Get(key string) ([]byte, bool) {
	var value []byte
	var ttlMin int
	var createdAt int64
	err := s.db.QueryRow("SELECT value, ttl_min, created_at FROM cache_entries WHERE key = ?", key).Scan(&value, &ttlMin, &createdAt)
	if err != nil {
		return nil, false
	}
	if time.Since(time.Unix(createdAt, 0)) > time.Duration(ttlMin)*time.Minute {
		s.db.Exec("DELETE FROM cache_entries WHERE key = ?", key)
		return nil, false
	}
	return value, true
}

// GetStale returns cached data even if TTL has expired (for fallback when FPB is down)
func (s *Store) GetStale(key string) ([]byte, bool) {
	var value []byte
	err := s.db.QueryRow("SELECT value FROM cache_entries WHERE key = ?", key).Scan(&value)
	if err != nil { return nil, false }
	return value, true
}

func (s *Store) Set(key string, value []byte, ttlMin int) error {
	_, err := s.db.Exec("INSERT OR REPLACE INTO cache_entries (key, value, ttl_min, created_at) VALUES (?, ?, ?, unixepoch())", key, value, ttlMin)
	return err
}

func (s *Store) Delete(key string) error {
	_, err := s.db.Exec("DELETE FROM cache_entries WHERE key = ?", key)
	return err
}

func (s *Store) Invalidate(prefix string) error {
	_, err := s.db.Exec("DELETE FROM cache_entries WHERE key LIKE ?", prefix+"%")
	return err
}

func (s *Store) Close() error { return s.db.Close() }
func (s *Store) Ping() bool { return s.db.Ping() == nil }
func (s *Store) DB() *sql.DB { return s.db }

// UpsertGame inserts or updates a game in the games table.
func (s *Store) UpsertGame(id, season, data, hora, equipaCasa, equipaFora, competicao, escalao, local, status, logoCasa, logoFora string, resultadoCasa, resultadoFora *int) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO games (id, season, data, hora, equipa_casa, equipa_fora, resultado_casa, resultado_fora, competicao, escalao, local, status, logo_casa, logo_fora, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, unixepoch())`,
		id, season, data, hora, equipaCasa, equipaFora, resultadoCasa, resultadoFora, competicao, escalao, local, status, logoCasa, logoFora)
	return err
}

// GetGamesBySeason returns all finished games for a season, ordered by date.
func (s *Store) GetGamesBySeason(season string) ([]GameRow, error) {
	// Debug: run raw query and log
	var total, withScores int
	s.db.QueryRow("SELECT COUNT(*) FROM games WHERE season = ?", season).Scan(&total)
	s.db.QueryRow("SELECT COUNT(*) FROM games WHERE season = ? AND resultado_casa IS NOT NULL AND resultado_fora IS NOT NULL", season).Scan(&withScores)
	
	// Also check: what seasons exist + grand total
	var grandTotal int
	var sampleSeason string
	s.db.QueryRow("SELECT COUNT(*) FROM games").Scan(&grandTotal)
	s.db.QueryRow("SELECT season FROM games LIMIT 1").Scan(&sampleSeason)
	log.Printf("[games] season=%q total=%d scored=%d grandTotal=%d sample=%q", season, total, withScores, grandTotal, sampleSeason)

	rows, err := s.db.Query(`SELECT id, season, data, equipa_casa, equipa_fora, resultado_casa, resultado_fora
		FROM games WHERE season = ? AND resultado_casa IS NOT NULL AND resultado_fora IS NOT NULL
		ORDER BY data ASC`, season)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []GameRow
	for rows.Next() {
		var g GameRow
		if err := rows.Scan(&g.ID, &g.Season, &g.Data, &g.HomeTeam, &g.AwayTeam, &g.HomeScore, &g.AwayScore); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// GameRow is a minimal game record for ELO calculation.
type GameRow struct {
	ID        string
	Season    string
	Data      string
	HomeTeam  string
	AwayTeam  string
	HomeScore int
	AwayScore int
}

func (s *Store) SaveMetric(ts time.Time, requests, cacheHits, cacheMisses, fpbRequests, rateLimited uint64, goroutines int) {
	s.db.Exec(`INSERT INTO metrics_snapshots (time, requests, cache_hits, cache_misses, fpb_requests, rate_limited, goroutines) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ts.Unix(), requests, cacheHits, cacheMisses, fpbRequests, rateLimited, goroutines)
}

type MetricRow struct {
	Time        int64
	Requests    int64
	CacheHits   int64
	CacheMisses int64
	FPBRequests int64
	RateLimited int64
	Goroutines  int64
}

func (s *Store) LoadMetrics(since time.Time, limit int) []MetricRow {
	rows, err := s.db.Query(`SELECT time, requests, cache_hits, cache_misses, fpb_requests, rate_limited, goroutines FROM metrics_snapshots WHERE time >= ? ORDER BY time ASC LIMIT ?`, since.Unix(), limit)
	if err != nil { return nil }
	defer rows.Close()
	var result []MetricRow
	for rows.Next() {
		var r MetricRow
		if err := rows.Scan(&r.Time, &r.Requests, &r.CacheHits, &r.CacheMisses, &r.FPBRequests, &r.RateLimited, &r.Goroutines); err == nil {
			result = append(result, r)
		}
	}
	return result
}

func (s *Store) PruneMetrics(before time.Time) {
	s.db.Exec("DELETE FROM metrics_snapshots WHERE time < ?", before.Unix())
}

func CacheKey(parts ...string) string {
	key := ""
	for i, p := range parts {
		if i > 0 {
			key += ":"
		}
		key += p
	}
	return key
}
