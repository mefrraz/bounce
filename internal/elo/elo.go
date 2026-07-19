package elo

import (
	"math"
	"sort"
)

const (
	DefaultELO = 1500.0
	KFactor    = 32.0
)

// Game represents a single game for ELO calculation.
type Game struct {
	HomeTeam  string
	AwayTeam  string
	HomeScore int
	AwayScore int
}

// Rating holds the ELO rating for a team.
type Rating struct {
	Team   string  `json:"team"`
	Rating float64 `json:"rating"`
}

// Calculate computes ELO ratings for a season given the list of games.
// All teams start at DefaultELO. Games should be in chronological order.
func Calculate(games []Game) []Rating {
	ratings := make(map[string]float64)

	getRating := func(team string) float64 {
		if r, ok := ratings[team]; ok {
			return r
		}
		ratings[team] = DefaultELO
		return DefaultELO
	}

	for _, g := range games {
		ra := getRating(g.HomeTeam)
		rb := getRating(g.AwayTeam)

		// Expected scores
		ea := 1.0 / (1.0 + math.Pow(10, (rb-ra)/400.0))
		eb := 1.0 - ea

		// Point differential multiplier (diminishing returns)
		margin := math.Abs(float64(g.HomeScore - g.AwayScore))
		marginMultiplier := math.Sqrt(margin+1) / 2.0
		if marginMultiplier > 1.5 {
			marginMultiplier = 1.5
		}

		var sa, sb float64
		if g.HomeScore > g.AwayScore {
			sa, sb = 1.0, 0.0
		} else if g.AwayScore > g.HomeScore {
			sa, sb = 0.0, 1.0
		} else {
			sa, sb = 0.5, 0.5
		}

		ratings[g.HomeTeam] = ra + KFactor*marginMultiplier*(sa-ea)
		ratings[g.AwayTeam] = rb + KFactor*marginMultiplier*(sb-eb)
	}

	// Convert map to sorted slice
	result := make([]Rating, 0, len(ratings))
	for team, rating := range ratings {
		result = append(result, Rating{Team: team, Rating: math.Round(rating)})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Rating > result[j].Rating
	})
	return result
}
