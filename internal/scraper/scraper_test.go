package scraper

import "testing"

func TestParseDatePt(t *testing.T) {
	tests := []struct{ in, want string }{
		{"18 de Junho de 2025", "2025-06-18"},
		{"1 de Janeiro de 2025", "2025-01-01"},
		{"31 de Dezembro de 2024", "2024-12-31"},
		{"15/03/2025", "2025-03-15"},
	}
	for _, tt := range tests {
		if got := ParseDatePt(tt.in); got != tt.want {
			t.Errorf("ParseDatePt(%q)=%q want %q", tt.in, got, tt.want)
		}
	}
}

func TestScrapeStandingsEmpty(t *testing.T) {
	if len(ScrapeStandings("")) != 0 { t.Error("expected empty") }
}

func TestScrapeGamesEmpty(t *testing.T) {
	if len(ScrapeGames("", "AGENDADO")) != 0 { t.Error("expected empty") }
}

func TestExtractFaseIDs(t *testing.T) {
	html := `<li class="option" tag="Fase Regular" value="30969"></li>`
	fases := ExtractFaseIDs(html)
	if len(fases) != 1 || fases[0].ID != "30969" { t.Errorf("got %v", fases) }
}

func TestScrapeGamesWithScore(t *testing.T) {
	html := `<div class="day-wrapper"><h3 class="date">4 OUT 2025</h3>
<a href="/ficha-de-jogo?internalID=413420" class="game-wrapper-a">
<div class="team-container"><span class="sigla">SLB</span><div class="image-container"><img src="a.png"/></div></div>
<div class="team-container right"><span class="sigla">FCP</span><div class="image-container"><img src="b.png"/></div></div>
<div class="results_wrapper row"><h3 class="results_text">89</h3><h3 class="results_text">73</h3></div>
<div class="location-wrapper"><span class="wrapper"><span><b>Pav</b></span></span><div class="competition"><span>Senior|Liga</span></div></div>
</a></div>`
	games := ScrapeGames(html, "FINALIZADO")
	if len(games) != 1 { t.Fatalf("expected 1, got %d", len(games)) }
	g := games[0]
	if g.ID != "413420" { t.Errorf("id: %s", g.ID) }
	if *g.HomeScore != 89 { t.Errorf("hs: %d", *g.HomeScore) }
	if *g.AwayScore != 73 { t.Errorf("as: %d", *g.AwayScore) }
	if g.HomeTeam != "SLB" { t.Errorf("home: %s", g.HomeTeam) }
	if g.AwayTeam != "FCP" { t.Errorf("away: %s", g.AwayTeam) }
	if g.Category != "Senior" { t.Errorf("cat: %s", g.Category) }
	if g.Competition != "Liga" { t.Errorf("comp: %s", g.Competition) }
}

func TestScrapeAthlete(t *testing.T) {
	html := `<div class="athleteDetailHighlight">
<div class="image"><img src="f.png"/></div><div class="number">13</div>
<div class="name">TM</div><div class="position">B</div><div class="club">FCG</div></div>
<table><tr><td>Jogos</td><td>10</td></tr><tr><td>Pontos</td><td>50</td></tr></table>`
	a := ScrapeAthlete(html)
	if a == nil { t.Fatal("nil") }
	if a.Name != "TM" { t.Errorf("name: %s", a.Name) }
	if a.Number != "13" { t.Errorf("num: %s", a.Number) }
	if a.Points != 50 { t.Errorf("pts: %d", a.Points) }
}

func TestSlugify(t *testing.T) {
	if s := slugify("FC Gaia"); s != "fc-gaia" { t.Errorf("got %s", s) }
}

func TestAtoi(t *testing.T) {
	if atoi("42") != 42 { t.Error("42") }
	if atoi("") != 0 { t.Error("empty") }
	if atoi("-") != 0 { t.Error("dash") }
}
