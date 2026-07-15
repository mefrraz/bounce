// Package cache provides SQLite-backed caching for scraped FPB data.
package cache

const (
	TTLLiveGame   = 2
	TTLToday      = 30
	TTLRecent     = 120
	TTLHistorical = 1440
	TTLStandings  = 60
)
