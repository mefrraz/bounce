package fpbapi

import (
	"log"
	"math"
)

// RecalculateELO is a placeholder for the full ELO recalculation.
// TODO: fetch games from Supabase for the season, compute ELO, store in SQLite.
func (f *FPBAPI) RecalculateELO(season string) error {
	log.Printf("[elo] recalculate %s: not yet implemented (needs Supabase game fetch)", season)
	// Placeholder — will fetch from Supabase in Phase 2
	_ = season
	_ = math.Pow
	return nil
}
