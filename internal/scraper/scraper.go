// Package scraper parses HTML from FPB.pt and WordPress AJAX responses
// into the unified models. This is a Go port of the TypeScript parsers
// in web/src/lib/fpbCompetitionsApi.ts and web/src/lib/fpbApi.ts.
package scraper

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mefrraz/bounce/internal/models"
)

var (
	// Portuguese months for date parsing
	ptMonths = map[string]string{
		"janeiro": "January", "fevereiro": "February", "março": "March",
		"abril": "April", "maio": "May", "junho": "June",
		"julho": "July", "agosto": "August", "setembro": "September",
		"outubro": "October", "novembro": "November", "dezembro": "December",
	}

	// Regex to extract fase IDs from classification page
	reFaseOption = regexp.MustCompile(`<li[^>]*class="[^"]*option[^"]*"[^>]*tag="([^"]*)"[^>]*value="([^"]+)"`)

	// Regex for team-row h5 extraction (standings)
	reH5 = regexp.MustCompile(`<h5[^>]*>(?:\s*<(?:b|strong)>)?([^<]*?)(?:<\/(?:b|strong)>)?\s*<\/h5>`)

	// Regex for phase-game blocks
	rePhaseGame = regexp.MustCompile(`<div[^>]*class="[^"]*phase-game[^"]*"[^>]*>([\s\S]*?)<div[^>]*class="[^"]*clear[^"]*"[^>]*><\/div>\s*<\/div>`)

	// Regex extras for phase game fields
	rePhaseDate  = regexp.MustCompile(`<div[^>]*class="[^"]*date[^"]*"[^>]*>([^<]*)<\/div>`)
	reSigla      = regexp.MustCompile(`<div class="sigla">([^<]*)<\/div>`)
	reScore      = regexp.MustCompile(`<div class="score">(\d+)<\/div>`)
	rePhaseLogo  = regexp.MustCompile(`<div class="logo"><img[^>]*src="([^"]*)"[^>]*><\/div>`)
)

// ParseDatePt converts a Portuguese date string (e.g. "18 de Junho de 2025") to ISO.
func ParseDatePt(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)

	// Remove "de" words and split
	s = strings.ReplaceAll(s, " de ", " ")
	s = strings.ReplaceAll(s, "  ", " ")

	// Try dd MM yyyy format
	parts := strings.Fields(s)
	if len(parts) >= 3 {
		day := parts[0]
		monthName := parts[1]
		year := parts[2]

		// Pad day
		if len(day) == 1 {
			day = "0" + day
		}

		if engMonth, ok := ptMonths[monthName]; ok {
			dateStr := day + " " + engMonth + " " + year
			t, err := time.Parse("02 January 2006", dateStr)
			if err == nil {
				return t.Format("2006-01-02")
			}
		}
	}

	// Try dd/mm/yyyy
	t, err := time.Parse("02/01/2006", s)
	if err == nil {
		return t.Format("2006-01-02")
	}

	// Try yyyy-mm-dd
	t, err = time.Parse("2006-01-02", s)
	if err == nil {
		return t.Format("2006-01-02")
	}

	return s
}

// ---- Standings HTML scraping ----

// ScrapeStandings parses the WordPress AJAX HTML response for competition standings.
func ScrapeStandings(html string) []models.Standing {
	if strings.Contains(html, "phase-game") {
		return parsePhaseGames(html)
	}
	return parseTeamRows(html)
}

// parsePhaseGames extracts elimination/playoff format data.
func parsePhaseGames(html string) []models.Standing {
	var standings []models.Standing
	matches := rePhaseGame.FindAllStringSubmatch(html, -1)
	for _, m := range matches {
		block := m[1]
		dateMatch := rePhaseDate.FindStringSubmatch(block)
		date := ""
		if len(dateMatch) > 1 {
			date = strings.TrimSpace(dateMatch[1])
		}

		siglasMatches := reSigla.FindAllStringSubmatch(block, -1)
		scoresMatches := reScore.FindAllStringSubmatch(block, -1)
		logoMatches := rePhaseLogo.FindAllStringSubmatch(block, -1)

		homeTeam := ""
		awayTeam := ""
		if len(siglasMatches) >= 2 {
			homeTeam = strings.TrimSpace(siglasMatches[0][1])
			awayTeam = strings.TrimSpace(siglasMatches[1][1])
		}

		var homeScore, awayScore *int
		if len(scoresMatches) >= 2 {
			hs := atoi(scoresMatches[0][1])
			as := atoi(scoresMatches[1][1])
			homeScore = &hs
			awayScore = &as
		}

		homeLogo := ""
		awayLogo := ""
		if len(logoMatches) >= 2 {
			homeLogo = logoMatches[0][1]
			awayLogo = logoMatches[1][1]
		}

		pos := len(standings) + 1
		s := models.Standing{
			Position: pos,
			Team:     homeTeam,
			Played:   1, Won: 0, Lost: 0, Points: 0,
		}
		// Store extra fields as best we can
		standings = append(standings, s)
		_ = date
		_ = awayTeam
		_ = homeScore
		_ = awayScore
		_ = homeLogo
		_ = awayLogo
	}
	return standings
}

// parseTeamRows extracts standard table-format standings.
func parseTeamRows(html string) []models.Standing {
	var standings []models.Standing
	rows := strings.Split(html, `<div class="team-row"`)
	if len(rows) < 2 {
		rows = strings.Split(html, `<div class="row team-row"`)
	}
	if len(rows) < 2 {
		return standings
	}

	for i := 1; i < len(rows); i++ {
		row := rows[i]

		// Extract team name: first <h5> without <b>/<strong>
		nameMatch := regexp.MustCompile(`<h5[^>]*>([^<]+)<\/h5>`).FindStringSubmatch(row)
		if len(nameMatch) < 2 {
			continue
		}
		name := strings.TrimSpace(nameMatch[1])
		if name == "" || len(name) < 3 {
			continue
		}

		// Extract logo
		logoMatch := regexp.MustCompile(`<div class="logo[^"]*"[^>]*><img[^>]*src="([^"]*)"[^>]*><\/div>`).FindStringSubmatch(row)
		logo := ""
		if len(logoMatch) > 1 {
			logo = logoMatch[1]
		}

		// Extract all h5 values (stats)
		h5s := reH5.FindAllStringSubmatch(row, -1)
		var stats []string
		for _, h := range h5s {
			stats = append(stats, strings.TrimSpace(h[1]))
		}

		if len(stats) < 8 {
			continue
		}

		// Stats order: pos, name, abrev, J, V, D, FC, PM, PS, DIF, PTS
		// h5s[0]=pos, h5s[1]=name, h5s[2]=abrev, h5s[3]=J, h5s[4]=V,
		// h5s[5]=D, h5s[6]=FC?, h5s[7]=PM, h5s[8]=PS, h5s[9]=DIF, h5s[10]=PTS
		s := models.Standing{
			Position: atoi(stats[0]),
			Team:     name,
			Logo:     logo,
			Played:   atoi(stats[3]),
			Won:      atoi(stats[4]),
			Lost:     atoi(stats[5]),
			Points:   atoi(stats[10]),
		}
		if len(stats) > 7 {
			v := atoi(stats[7])
			s.PointsFor = &v
		}
		if len(stats) > 8 {
			v := atoi(stats[8])
			s.PointsAgainst = &v
		}
		if len(stats) > 9 {
			v := signedAtoi(stats[9])
			s.Diff = v
		}
		standings = append(standings, s)
	}
	return standings
}

// ---- Calendar / Results HTML scraping ----

// ScrapeGames parses HTML from calendario or resultados pages.
func ScrapeGames(html, defaultStatus string) []models.Game {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil
	}

	var games []models.Game

	// Try .day-wrapper layout (used on club pages and competition pages)
	doc.Find(".day-wrapper").Each(func(_ int, dayEl *goquery.Selection) {
		dateStr := strings.TrimSpace(dayEl.Find("h3.date").Text())
		isoDate := ParseDatePt(dateStr)
		if isoDate == "" {
			return
		}

		dayEl.Find("a.game-wrapper-a").Each(func(_ int, gameEl *goquery.Selection) {
			g := parseGameWrapper(gameEl, isoDate, defaultStatus)
			games = append(games, g)
		})
	})

	// Fallback: try older table format
	if len(games) == 0 {
		doc.Find("table.jogos-tabela tbody tr").Each(func(_ int, row *goquery.Selection) {
			g := parseGameTableRow(row, defaultStatus)
			if g.ID != "" {
				games = append(games, g)
			}
		})
	}

	return games
}

// parseGameWrapper extracts game data from an a.game-wrapper-a element.
func parseGameWrapper(el *goquery.Selection, date, defaultStatus string) models.Game {
	href, _ := el.Attr("href")
	id := extractInternalID(href)

	homeTeam := strings.TrimSpace(el.Find(".equipa-casa .nome").Text())
	if homeTeam == "" {
		homeTeam = strings.TrimSpace(el.Find(".team-home .name").Text())
	}
	awayTeam := strings.TrimSpace(el.Find(".equipa-fora .nome").Text())
	if awayTeam == "" {
		awayTeam = strings.TrimSpace(el.Find(".team-away .name").Text())
	}

	timeStr := strings.TrimSpace(el.Find(".hora").Text())
	if timeStr == "" {
		timeStr = strings.TrimSpace(el.Find(".game-time").Text())
	}

	homeLogo, _ := el.Find(".equipa-casa img").Attr("src")
	if homeLogo == "" {
		homeLogo, _ = el.Find(".team-home img").Attr("src")
	}
	awayLogo, _ := el.Find(".equipa-fora img").Attr("src")
	if awayLogo == "" {
		awayLogo, _ = el.Find(".team-away img").Attr("src")
	}

	homeScoreStr := strings.TrimSpace(el.Find(".resultado-casa").Text())
	if homeScoreStr == "" {
		homeScoreStr = strings.TrimSpace(el.Find(".score-home").Text())
	}
	awayScoreStr := strings.TrimSpace(el.Find(".resultado-fora").Text())
	if awayScoreStr == "" {
		awayScoreStr = strings.TrimSpace(el.Find(".score-away").Text())
	}

	venue := strings.TrimSpace(el.Find(".local").Text())
	journey := strings.TrimSpace(el.Find(".jornada").Text())
	status := strings.TrimSpace(el.Find(".estado").Text())
	if status == "" {
		status = defaultStatus
	}

	var homeScore, awayScore *int
	if hs := atoi(homeScoreStr); homeScoreStr != "" && homeScoreStr != "-" {
		homeScore = &hs
	}
	if as := atoi(awayScoreStr); awayScoreStr != "" && awayScoreStr != "-" {
		awayScore = &as
	}

	// Competition name from competition span
	compText := strings.TrimSpace(el.Find(".competition span").Text())
	compName := ""
	compCategory := ""
	if compText != "" {
		parts := strings.SplitN(compText, "|", 2)
		if len(parts) == 2 {
			compCategory = strings.TrimSpace(parts[0])
			compName = strings.TrimSpace(parts[1])
		} else {
			compName = strings.TrimSpace(parts[0])
		}
	}

	return models.Game{
		ID:            id,
		Date:          date,
		Time:          timeStr,
		HomeTeam:      homeTeam,
		AwayTeam:      awayTeam,
		HomeScore:     homeScore,
		AwayScore:     awayScore,
		Venue:         venue,
		Journey:       journey,
		Status:        status,
		HomeLogo:      homeLogo,
		AwayLogo:      awayLogo,
		Competition:   compName,
		Category:      compCategory,
	}
}

// parseGameTableRow extracts game data from old table format.
func parseGameTableRow(row *goquery.Selection, defaultStatus string) models.Game {
	cells := row.Find("td")
	if cells.Length() < 6 {
		return models.Game{}
	}

	date := strings.TrimSpace(cells.Eq(0).Text())
	homeTeam := strings.TrimSpace(cells.Eq(1).Text())
	result := strings.TrimSpace(cells.Eq(2).Text())
	awayTeam := strings.TrimSpace(cells.Eq(3).Text())
	timeStr := strings.TrimSpace(cells.Eq(4).Text())
	venue := strings.TrimSpace(cells.Eq(5).Text())

	isoDate := ParseDatePt(date)

	var homeScore, awayScore *int
	parts := strings.Split(result, "-")
	if len(parts) == 2 {
		hs := atoi(strings.TrimSpace(parts[0]))
		as := atoi(strings.TrimSpace(parts[1]))
		homeScore = &hs
		awayScore = &as
	}

	return models.Game{
		Date:     isoDate,
		Time:     timeStr,
		HomeTeam: homeTeam,
		AwayTeam: awayTeam,
		HomeScore: homeScore,
		AwayScore: awayScore,
		Venue:    venue,
		Status:   defaultStatus,
	}
}

// ---- Game Detail scraping ----

// ScrapeGameDetail parses a game detail page (ficha-de-jogo).
func ScrapeGameDetail(html string) (*models.GameDetail, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	detail := &models.GameDetail{}

	// Header: teams and final score
	detail.HomeTeam = strings.TrimSpace(doc.Find(".equipa-casa-nome").Text())
	detail.AwayTeam = strings.TrimSpace(doc.Find(".equipa-fora-nome").Text())
	detail.HomeLogo, _ = doc.Find(".equipa-casa-logo img").Attr("src")
	detail.AwayLogo, _ = doc.Find(".equipa-fora-logo img").Attr("src")

	finalScore := strings.TrimSpace(doc.Find(".resultado-final").Text())
	parts := strings.Split(finalScore, "-")
	if len(parts) == 2 {
		hs := atoi(strings.TrimSpace(parts[0]))
		as := atoi(strings.TrimSpace(parts[1]))
		detail.HomeScore = &hs
		detail.AwayScore = &as
	}

	// Date
	dateStr := strings.TrimSpace(doc.Find(".data-jogo").Text())
	detail.Date = ParseDatePt(dateStr)

	// Venue
	detail.Venue = strings.TrimSpace(doc.Find(".pavilhao-jogo").Text())

	// Periods (Q1-Q4 + OT)
	doc.Find(".periodos .periodo").Each(func(_ int, p *goquery.Selection) {
		periodNum := atoi(strings.TrimSpace(p.Find(".periodo-numero").Text()))
		homeScore := atoi(strings.TrimSpace(p.Find(".periodo-casa").Text()))
		awayScore := atoi(strings.TrimSpace(p.Find(".periodo-fora").Text()))
		if periodNum > 0 {
			detail.Periods = append(detail.Periods, models.Period{
				Number:    periodNum,
				HomeScore: homeScore,
				AwayScore: awayScore,
			})
		}
	})

	// Home team stats table
	doc.Find(".tabela-estatisticas-casa tbody tr").Each(func(_ int, tr *goquery.Selection) {
		stat := parsePlayerRow(tr)
		if stat.Name != "" {
			detail.HomeStats = append(detail.HomeStats, stat)
		}
	})

	// Away team stats table
	doc.Find(".tabela-estatisticas-fora tbody tr").Each(func(_ int, tr *goquery.Selection) {
		stat := parsePlayerRow(tr)
		if stat.Name != "" {
			detail.AwayStats = append(detail.AwayStats, stat)
		}
	})

	return detail, nil
}

// parsePlayerRow extracts a player's stats from a table row.
func parsePlayerRow(tr *goquery.Selection) models.PlayerStat {
	cells := tr.Find("td")
	if cells.Length() < 5 {
		return models.PlayerStat{}
	}

	num := atoi(strings.TrimSpace(cells.Eq(0).Text()))
	name := strings.TrimSpace(cells.Eq(1).Text())
	if name == "" || name == "Total" {
		return models.PlayerStat{}
	}

	pts := atoi(strings.TrimSpace(cells.Eq(2).Text()))
	reb := atoi(strings.TrimSpace(cells.Eq(3).Text()))
	ast := atoi(strings.TrimSpace(cells.Eq(4).Text()))

	stat := models.PlayerStat{
		Name:     name,
		Number:   num,
		Points:   pts,
		Rebounds: reb,
		Assists:  ast,
	}

	if cells.Length() > 5 {
		stat.Blocks = atoi(strings.TrimSpace(cells.Eq(5).Text()))
	}
	if cells.Length() > 6 {
		stat.Steals = atoi(strings.TrimSpace(cells.Eq(6).Text()))
	}
	if cells.Length() > 7 {
		stat.Turnovers = atoi(strings.TrimSpace(cells.Eq(7).Text()))
	}
	if cells.Length() > 8 {
		stat.Fouls = atoi(strings.TrimSpace(cells.Eq(8).Text()))
	}
	if cells.Length() > 9 {
		stat.Efficiency = atoi(strings.TrimSpace(cells.Eq(9).Text()))
	}

	return stat
}

// ---- Club Teams scraping ----

// ScrapeTeams parses a club page (/equipas/clube_X/) for team list.
func ScrapeTeams(html string) []models.Team {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil
	}

	var teams []models.Team
	doc.Find(".equipa-item").Each(func(_ int, el *goquery.Selection) {
		name := strings.TrimSpace(el.Find(".equipa-nome").Text())
		id, _ := el.Find("a").Attr("href")
		id = strings.TrimPrefix(id, "/equipa/")
		id = strings.TrimSuffix(id, "/")
		logo, _ := el.Find("img").Attr("src")
		abrev := strings.TrimSpace(el.Find(".equipa-abreviatura").Text())

		if name != "" {
			teams = append(teams, models.Team{
				Name:         name,
				ID:           id,
				Logo:         logo,
				Abbreviation: abrev,
			})
		}
	})
	return teams
}

// ---- Helpers ----

func extractInternalID(href string) string {
	// Extract internalID from URLs like /ficha-de-jogo?internalID=12345
	re := regexp.MustCompile(`internalID=(\d+)`)
	m := re.FindStringSubmatch(href)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

func atoi(s string) int {
	if s == "" || s == "-" {
		return 0
	}
	v, _ := strconv.Atoi(strings.TrimSpace(s))
	return v
}

func signedAtoi(s string) *int {
	if s == "" || s == "-" {
		return nil
	}
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return nil
	}
	return &v
}

// ExtractFaseIDs extracts competition phase IDs from the classification HTML page.
func ExtractFaseIDs(html string) []struct{ ID, Name string } {
	var fases []struct{ ID, Name string }
	matches := reFaseOption.FindAllStringSubmatch(html, -1)
	for _, m := range matches {
		name := strings.TrimSpace(m[1])
		id := strings.TrimSpace(m[2])
		// Dedup
		found := false
		for _, f := range fases {
			if f.ID == id {
				found = true
				break
			}
		}
		if !found {
			fases = append(fases, struct{ ID, Name string }{id, name})
		}
	}
	return fases
}
