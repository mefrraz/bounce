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
	HomeTeam   string
	AwayTeam   string
	HomeScore  int
	AwayScore  int
	HomePriority int // club priority level (1-4, lower = higher division)
	AwayPriority int
}

// TeamRating holds intermediate ELO rating for a team name during calculation.
type TeamRating struct {
	Team        string
	Rating      float64
	GamesPlayed int
	Priority    int
}

// Calculate computes per-season ELO from a chronologically sorted list of games.
// Teams start at DefaultELO. Priority adjustment: (awayPrio - homePrio) * 100 points.
func Calculate(games []Game) []TeamRating {
	ratings := make(map[string]*TeamRating)

	get := func(team string, prio int) *TeamRating {
		if r, ok := ratings[team]; ok { return r }
		r := &TeamRating{Team: team, Rating: DefaultELO, Priority: prio}
		ratings[team] = r
		return r
	}

	for _, g := range games {
		home := get(g.HomeTeam, g.HomePriority)
		away := get(g.AwayTeam, g.AwayPriority)

		// Priority adjusted expected score
		priorityAdj := float64(g.AwayPriority-g.HomePriority) * 100.0
		eHome := 1.0 / (1.0 + math.Pow(10, (away.Rating-home.Rating+priorityAdj)/400.0))
		eAway := 1.0 - eHome

		var sHome, sAway float64
		if g.HomeScore > g.AwayScore {
			sHome, sAway = 1, 0
		} else if g.AwayScore > g.HomeScore {
			sHome, sAway = 0, 1
		} else {
			sHome, sAway = 0.5, 0.5
		}

		home.Rating += KFactor * (sHome - eHome)
		away.Rating += KFactor * (sAway - eAway)
		home.GamesPlayed++
		away.GamesPlayed++
	}

	result := make([]TeamRating, 0, len(ratings))
	for _, r := range ratings {
		result = append(result, *r)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Rating > result[j].Rating })
	return result
}
