package elo

import "math"

const DefaultRating = 1500
const KFactor = 32
const HomeAdvantage = 50

type Rating struct {
	TeamID string  `json:"team_id"`
	Team   string  `json:"team"`
	Rating float64 `json:"rating"`
	Games  int     `json:"games"`
}

type Outcome struct {
	WinnerID   string
	LoserID    string
	WinnerName string
	LoserName  string
	HomeTeamID string
	ScoreDiff  int
}

func UpdateRatings(o Outcome, ratings map[string]*Rating) (float64, float64) {
	w := ratings[o.WinnerID]
	if w == nil {
		w = &Rating{TeamID: o.WinnerID, Team: o.WinnerName, Rating: DefaultRating}
		ratings[o.WinnerID] = w
	}
	l := ratings[o.LoserID]
	if l == nil {
		l = &Rating{TeamID: o.LoserID, Team: o.LoserName, Rating: DefaultRating}
		ratings[o.LoserID] = l
	}
	ew := w.Rating
	el2 := l.Rating
	if o.HomeTeamID == o.WinnerID {
		ew += HomeAdvantage
	} else {
		el2 += HomeAdvantage
	}
	expW := 1.0 / (1.0 + math.Pow(10, (el2-ew)/400))
	margin := marginFactor(o.ScoreDiff)
	dw := KFactor * margin * (1.0 - expW)
	dl := KFactor * margin * (0.0 - (1.0 - expW))
	w.Rating += dw
	l.Rating += dl
	w.Games++
	l.Games++
	return dw, dl
}

func marginFactor(diff int) float64 {
	if diff < 0 {
		diff = -diff
	}
	if diff <= 10 {
		return 1.0
	}
	if diff <= 20 {
		return 1.1
	}
	return 1.25
}

func PredictWinProbability(ratingA, ratingB float64, homeA bool) float64 {
	ea := ratingA
	eb := ratingB
	if homeA {
		ea += HomeAdvantage
	} else {
		eb += HomeAdvantage
	}
	return 1.0 / (1.0 + math.Pow(10, (eb-ea)/400))
}
