package cache

import (
	"fmt"
	"time"
)

// IsToday returns true if the date string (any format parseable by ParseDatePt)
// represents today in the local timezone.
func IsToday(dateStr string) bool {
	if dateStr == "" { return false }
	t, err := parseDate(dateStr)
	if err != nil { return false }
	today := time.Now().Truncate(24 * time.Hour)
	return t.Truncate(24 * time.Hour).Equal(today)
}

// IsOffSeason returns true during summer months (June, July, August)
// when no basketball games are typically scheduled.
func IsOffSeason() bool {
	m := time.Now().Month()
	return m >= time.June && m <= time.August
}

// TTLForGame returns the appropriate TTL in minutes for a game
// based on its date and status.
func TTLForGame(dateStr, status string) int {
	if !IsToday(dateStr) {
		return 1440 // 24h for historical games
	}

	switch status {
	case "AGENDADO", "AO VIVO", "EM CURSO":
		return 2
	case "FINALIZADO":
		// Check if the game finished recently (<1 hour ago)
		t, _ := parseDate(dateStr)
		if time.Since(t) < 1*time.Hour {
			return 5 // still might have updates (stats, periods)
		}
		return 1440 // historical, done
	default:
		// Future game more than 2 days away
		t, _ := parseDate(dateStr)
		if t.After(time.Now().Add(48 * time.Hour)) {
			return 360 // 6h for far-future games
		}
		return 2 // today/tomorrow game
	}
}

// TTLForCalendar returns the TTL for a club's game list based on
// whether there are games today and whether it's off-season.
func TTLForCalendar(hasGamesToday bool, isOffSeason bool) int {
	if isOffSeason {
		return 1440 // 24h during summer
	}
	if hasGamesToday {
		return 2 // re-check every 2 min when games are on
	}
	return 360 // 6h during season when no games today
}

// parseDate tries multiple date formats common in FPB data.
func parseDate(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02",
		"02 Jan 2006",
		"2 Jan 2006",
		"02/01/2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, nil
}

// CurrentSeason returns the current basketball season string (YYYY/YYYY).
// The season starts in September.
func CurrentSeason() string {
	now := time.Now()
	if now.Month() >= time.September {
		return fmt.Sprintf("%d/%d", now.Year(), now.Year()+1)
	}
	return fmt.Sprintf("%d/%d", now.Year()-1, now.Year())
}
