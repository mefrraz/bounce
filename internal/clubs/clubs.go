package clubs

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed clubs.json
var clubsJSON []byte

type Club struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	Slug          string  `json:"slug"`
	ShortName     string  `json:"short_name"`
	SearchName    string  `json:"search_name"`
	LogoURL       string  `json:"logo_url"`
	PrimaryColor  string  `json:"primary_color,omitempty"`
	LogoSecondary string  `json:"logo_secondary,omitempty"`
	Priority      int     `json:"priority,omitempty"`
	EloRating     float64 `json:"elo_rating,omitempty"`
	LogoPattern   string  `json:"-"` // computed: last path segment of logo URL
}

var clubs []Club
var byLogo map[string]*Club

func init() {
	json.Unmarshal(clubsJSON, &clubs)
	byLogo = make(map[string]*Club)
	for i := range clubs {
		c := &clubs[i]
		// Extract logo pattern: last path segment before extension
		if c.LogoURL != "" {
			parts := strings.Split(c.LogoURL, "/")
			last := parts[len(parts)-1]
			if dot := strings.LastIndex(last, "."); dot > 0 {
				last = last[:dot]
			}
			c.LogoPattern = strings.ToLower(last)
			byLogo[c.LogoPattern] = c
		}
	}
}

func All() []Club { return clubs }

// MatchByLogo tries to find a club whose logo URL pattern appears in the given URL.
// If found, returns canonical name, slug, and logo URL.
func MatchByLogo(logoURL string) *Club {
	if logoURL == "" {
		return nil
	}
	parts := strings.Split(logoURL, "/")
	last := parts[len(parts)-1]
	if dot := strings.LastIndex(last, "."); dot > 0 {
		last = last[:dot]
	}
	pattern := strings.ToLower(last)
	if c, ok := byLogo[pattern]; ok {
		return c
	}
	// Fuzzy: check if any known pattern is contained in the URL
	for _, c := range clubs {
		if c.LogoPattern != "" && strings.Contains(strings.ToLower(logoURL), c.LogoPattern) {
			return &c
		}
	}
	return nil
}

// NormalizeTeam returns canonical short_name and logo for a team, given the raw name and logo URL from FPB.
func NormalizeTeam(rawName, logoURL string) (string, string) {
	if c := MatchByLogo(logoURL); c != nil {
		name := c.ShortName
		if name == "" {
			name = c.Name
		}
		return name, c.LogoURL
	}
	return rawName, logoURL
}
