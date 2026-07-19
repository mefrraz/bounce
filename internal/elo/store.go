package elo

import (
	"database/sql"
	"time"
)

// Store provides access to the elo_history table in the shared SQLite DB.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// RatingRow mirrors the elo_history table.
type RatingRow struct {
	ClubID      int     `json:"club_id"`
	Season      string  `json:"season"`
	EloRating   float64 `json:"elo_rating"`
	GamesPlayed int     `json:"games_played"`
}

// Upsert writes a rating row (insert or replace).
func (s *Store) Upsert(r RatingRow) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO elo_history (club_id, season, elo_rating, games_played, updated_at)
		 VALUES (?, ?, ?, ?, unixepoch())`,
		r.ClubID, r.Season, r.EloRating, r.GamesPlayed,
	)
	return err
}

// BatchUpsert writes multiple ratings in a single transaction.
func (s *Store) BatchUpsert(rows []RatingRow) error {
	tx, err := s.db.Begin()
	if err != nil { return err }
	defer tx.Rollback()
	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO elo_history (club_id, season, elo_rating, games_played, updated_at) VALUES (?, ?, ?, ?, unixepoch())`)
	if err != nil { return err }
	defer stmt.Close()
	for _, r := range rows {
		if _, err := stmt.Exec(r.ClubID, r.Season, r.EloRating, r.GamesPlayed); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetSeason returns all ELO ratings for a given season, ordered by rating desc.
func (s *Store) GetSeason(season string) ([]RatingRow, error) {
	rows, err := s.db.Query(
		`SELECT club_id, season, elo_rating, games_played FROM elo_history WHERE season = ? ORDER BY elo_rating DESC`,
		season,
	)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []RatingRow
	for rows.Next() {
		var r RatingRow
		if err := rows.Scan(&r.ClubID, &r.Season, &r.EloRating, &r.GamesPlayed); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// HasSeason checks if any ELO data exists for a season.
func (s *Store) HasSeason(season string) bool {
	var ok int
	s.db.QueryRow("SELECT 1 FROM elo_history WHERE season = ? LIMIT 1", season).Scan(&ok)
	return ok == 1
}

// LastUpdate returns the most recent updated_at for a season.
func (s *Store) LastUpdate(season string) (time.Time, bool) {
	var ts int64
	err := s.db.QueryRow("SELECT MAX(updated_at) FROM elo_history WHERE season = ?", season).Scan(&ts)
	if err != nil || ts == 0 {
		return time.Time{}, false
	}
	return time.Unix(ts, 0), true
}
