package cache

import (
	"testing"
	"time"
)

func TestIsToday(t *testing.T) {
	today := time.Now().Format("2 Jan 2006")
	if !IsToday(today) { t.Error("IsToday should be true for today") }
	if IsToday("1 Jan 2020") { t.Error("IsToday should be false for past date") }
	if IsToday("") { t.Error("IsToday should be false for empty string") }
}

func TestIsOffSeason(t *testing.T) {
	// Can't test precisely without mocking, just check it's a bool
	_ = IsOffSeason()
}

func TestTTLForGame(t *testing.T) {
	today := time.Now().Format("2 Jan 2006")
	yesterday := time.Now().Add(-24 * time.Hour).Format("2 Jan 2006")

	if ttl := TTLForGame(today, "AGENDADO"); ttl != 2 {
		t.Errorf("Today AGENDADO: expected 2, got %d", ttl)
	}
	if ttl := TTLForGame(today, "AO VIVO"); ttl != 2 {
		t.Errorf("Today AO VIVO: expected 2, got %d", ttl)
	}
	if ttl := TTLForGame(yesterday, "FINALIZADO"); ttl != 1440 {
		t.Errorf("Yesterday FINALIZADO: expected 1440, got %d", ttl)
	}
}

func TestTTLForCalendar(t *testing.T) {
	if ttl := TTLForCalendar(true, false); ttl != 2 {
		t.Errorf("Has games, in season: expected 2, got %d", ttl)
	}
	if ttl := TTLForCalendar(false, false); ttl != 360 {
		t.Errorf("No games, in season: expected 360, got %d", ttl)
	}
	if ttl := TTLForCalendar(false, true); ttl != 1440 {
		t.Errorf("No games, off season: expected 1440, got %d", ttl)
	}
}
