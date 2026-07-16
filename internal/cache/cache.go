package cache

import (
	"database/sql"
	"fmt"
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
