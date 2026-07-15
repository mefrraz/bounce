package scraper

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mefrraz/bounce/internal/models"
)

var ptMonths = map[string]string{
	"janeiro": "January", "fevereiro": "February", "marco": "March",
	"abril": "April", "maio": "May", "junho": "June",
	"julho": "July", "agosto": "August", "setembro": "September",
	"outubro": "October", "novembro": "November", "dezembro": "December",
}

func ParseDatePt(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, " de ", " ")
	parts := strings.Fields(s)
	if len(parts) >= 3 {
		if engMonth, ok := ptMonths[parts[1]]; ok {
			day := parts[0]
			if len(day) == 1 {
				day = "0" + day
			}
			t, err := time.Parse("02 January 2006", day+" "+engMonth+" "+parts[2])
			if err == nil {
				return t.Format("2006-01-02")
			}
		}
	}
	t, err := time.Parse("02/01/2006", s)
	if err == nil {
		return t.Format("2006-01-02")
	}
	return s
}

func ScrapeStandings(html string) []models.Standing {
	rows := strings.Split(html, `<div class="team-row"`)
	if len(rows) < 2 {
		rows = strings.Split(html, `<div class="row team-row"`)
	}
	var standings []models.Standing
	reH5 := regexp.MustCompile(`<h5[^>]*>(?:\s*<(?:b|strong)>)?([^<]*?)(?:<\/(?:b|strong)>)?\s*<\/h5>`)
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		nameMatch := regexp.MustCompile(`<h5[^>]*>([^<]+)<\/h5>`).FindStringSubmatch(row)
		if len(nameMatch) < 2 {
			continue
		}
		name := strings.TrimSpace(nameMatch[1])
		if len(name) < 3 {
			continue
		}
		h5s := reH5.FindAllStringSubmatch(row, -1)
		var stats []string
		for _, h := range h5s {
			stats = append(stats, strings.TrimSpace(h[1]))
		}
		if len(stats) < 8 {
			continue
		}
		logo := ""
		if lm := regexp.MustCompile(`<img[^>]*src="([^"]*)"`).FindStringSubmatch(row); len(lm) > 1 {
			logo = lm[1]
		}
		s := models.Standing{
			Position: atoi(stats[0]), Team: name, Logo: logo,
			Played: atoi(stats[3]), Won: atoi(stats[4]), Lost: atoi(stats[5]), Points: atoi(stats[10]),
		}
		if len(stats) > 7 {
			v := atoi(stats[7]); s.PointsFor = &v
		}
		if len(stats) > 8 {
			v := atoi(stats[8]); s.PointsAgainst = &v
		}
		standings = append(standings, s)
	}
	return standings
}

func ScrapeGames(html, defaultStatus string) []models.Game {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil
	}
	var games []models.Game
	doc.Find(".day-wrapper").Each(func(_ int, dayEl *goquery.Selection) {
		dateStr := strings.TrimSpace(dayEl.Find("h3.date").Text())
		isoDate := ParseDatePt(dateStr)
		if isoDate == "" {
			return
		}
		dayEl.Find("a.game-wrapper-a").Each(func(_ int, el *goquery.Selection) {
			href, _ := el.Attr("href")
			id := extractInternalID(href)
			homeTeam := strings.TrimSpace(el.Find(".equipa-casa .nome").Text())
			awayTeam := strings.TrimSpace(el.Find(".equipa-fora .nome").Text())
			timeStr := strings.TrimSpace(el.Find(".hora").Text())
			homeLogo, _ := el.Find(".equipa-casa img").Attr("src")
			awayLogo, _ := el.Find(".equipa-fora img").Attr("src")
			homeScoreStr := strings.TrimSpace(el.Find(".resultado-casa").Text())
			awayScoreStr := strings.TrimSpace(el.Find(".resultado-fora").Text())
			venue := strings.TrimSpace(el.Find(".local").Text())
			journey := strings.TrimSpace(el.Find(".jornada").Text())
			status := strings.TrimSpace(el.Find(".estado").Text())
			if status == "" {
				status = defaultStatus
			}
			var hs, as *int
			if v := atoi(homeScoreStr); homeScoreStr != "" && homeScoreStr != "-" {
				hs = &v
			}
			if v := atoi(awayScoreStr); awayScoreStr != "" && awayScoreStr != "-" {
				as = &v
			}
			games = append(games, models.Game{
				ID: id, Date: isoDate, Time: timeStr,
				HomeTeam: homeTeam, AwayTeam: awayTeam,
				HomeScore: hs, AwayScore: as,
				Venue: venue, Journey: journey, Status: status,
				HomeLogo: homeLogo, AwayLogo: awayLogo,
			})
		})
	})
	return games
}

func ScrapeGameDetail(html string) (*models.GameDetail, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	detail := &models.GameDetail{}

	// Teams
	detail.HomeTeam = strings.TrimSpace(doc.Find(".equipa-casa .nome").First().Text())
	if detail.HomeTeam == "" {
		detail.HomeTeam = strings.TrimSpace(doc.Find("h2.home-team").First().Text())
	}
	detail.AwayTeam = strings.TrimSpace(doc.Find(".equipa-fora .nome").First().Text())
	if detail.AwayTeam == "" {
		detail.AwayTeam = strings.TrimSpace(doc.Find("h2.away-team").First().Text())
	}
	detail.HomeLogo, _ = doc.Find(".equipa-casa img").First().Attr("src")
	detail.AwayLogo, _ = doc.Find(".equipa-fora img").First().Attr("src")

	// Score
	scoreStr := strings.TrimSpace(doc.Find(".resultado-final").First().Text())
	if scoreStr == "" {
		scoreStr = strings.TrimSpace(doc.Find(".final-score").First().Text())
	}
	parts := strings.Split(scoreStr, "-")
	if len(parts) == 2 {
		hs := atoi(strings.TrimSpace(parts[0]))
		as := atoi(strings.TrimSpace(parts[1]))
		detail.HomeScore = &hs
		detail.AwayScore = &as
	}

	// Date and venue
	detail.Date = ParseDatePt(strings.TrimSpace(doc.Find(".data-jogo").First().Text()))
	detail.Venue = strings.TrimSpace(doc.Find(".pavilhao-jogo").First().Text())
	detail.Status = "FINALIZADO"

	// Periods
	doc.Find("table.ficha-tabela tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() >= 4 {
			periodLabel := strings.TrimSpace(cells.Eq(0).Text())
			if strings.Contains(periodLabel, "Q") || strings.Contains(periodLabel, "P") {
				num := atoi(strings.TrimLeft(periodLabel, "QP"))
				hs := atoi(strings.TrimSpace(cells.Eq(1).Text()))
				as := atoi(strings.TrimSpace(cells.Eq(2).Text()))
				if num > 0 {
					detail.Periods = append(detail.Periods, models.Period{
						Number: num, HomeScore: hs, AwayScore: as,
					})
				}
			}
		}
	})

	return detail, nil
}

func ExtractFaseIDs(html string) []struct{ ID, Name string } {
	re := regexp.MustCompile(`<li[^>]*class="[^"]*option[^"]*"[^>]*tag="([^"]*)"[^>]*value="([^"]+)"`)
	var fases []struct{ ID, Name string }
	for _, m := range re.FindAllStringSubmatch(html, -1) {
		fases = append(fases, struct{ ID, Name string }{m[2], strings.TrimSpace(m[1])})
	}
	return fases
}

func extractInternalID(href string) string {
	m := regexp.MustCompile(`internalID=(\d+)`).FindStringSubmatch(href)
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

// ---- Athlete scraping ----

// AthleteData holds the parsed athlete profile from /atletas/{id}/
type AthleteData struct {
	Name           string `json:"nome"`
	Photo          string `json:"foto"`
	Number         string `json:"numero"`
	Position       string `json:"posicao"`
	Club           string `json:"clube"`
	Nationality    string `json:"nacionalidade"`
	FlagURL        string `json:"bandeira_url"`
	LicenseNumber  string `json:"nr_licenca"`
	BirthDate      string `json:"data_nascimento"`
	Games          int    `json:"jogos"`
	Points         int    `json:"pontos"`
	Rebounds       int    `json:"ressaltos"`
	Assists        int    `json:"assistencias"`
}

// ScrapeAthlete parses an athlete detail page.
func ScrapeAthlete(html string) *AthleteData {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil
	}

	a := &AthleteData{}

	highlight := doc.Find(".athleteDetailHighlight")
	a.Photo, _ = highlight.Find(".image img").Attr("src")
	a.Number = strings.TrimSpace(highlight.Find(".number").Text())
	a.Name = strings.TrimSpace(highlight.Find(".name").Text())
	a.Position = strings.TrimSpace(highlight.Find(".position").Text())
	a.Club = strings.TrimSpace(highlight.Find(".club").Text())

	// Stats table
	doc.Find("table tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 2 {
			return
		}
		label := strings.TrimSpace(cells.Eq(0).Text())
		val := strings.TrimSpace(cells.Eq(1).Text())
		switch {
		case strings.Contains(label, "Jogos"):
			a.Games = atoi(val)
		case strings.Contains(label, "Pontos"):
			a.Points = atoi(val)
		case strings.Contains(label, "Ressaltos"):
			a.Rebounds = atoi(val)
		case strings.Contains(label, "Assist"):
			a.Assists = atoi(val)
		}
	})

	return a
}

// ---- Team Detail scraping ----

// TeamDetail holds parsed team page data.
type TeamDetail struct {
	Name    string       `json:"nome"`
	Club    string       `json:"clube"`
	Photo   string       `json:"foto"`
	Players []TeamPlayer `json:"jogadores"`
	Games   []models.Game `json:"jogos"`
}

// TeamPlayer represents a player in the team roster.
type TeamPlayer struct {
	Name   string `json:"nome"`
	Number int    `json:"numero"`
	Photo  string `json:"foto"`
	ID     string `json:"atleta_id"`
}

// ScrapeTeamDetail parses a team page (/equipa/{id}/).
func ScrapeTeamDetail(html string) *TeamDetail {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil
	}

	td := &TeamDetail{}
	td.Name = strings.TrimSpace(doc.Find("h1").First().Text())
	td.Photo, _ = doc.Find(".team-photo img").First().Attr("src")
	td.Club = strings.TrimSpace(doc.Find(".club-name").First().Text())

	// Players
	doc.Find(".player-card, .jogador-item, table.plantel tbody tr").Each(func(_ int, el *goquery.Selection) {
		name := strings.TrimSpace(el.Find(".player-name, .nome").First().Text())
		photo, _ := el.Find("img").First().Attr("src")
		num := atoi(strings.TrimSpace(el.Find(".player-number, .numero").First().Text()))
		id, _ := el.Find("a").Attr("href")
		id = strings.TrimPrefix(id, "/atletas/")
		id = strings.TrimSuffix(id, "/")

		if name != "" {
			td.Players = append(td.Players, TeamPlayer{Name: name, Number: num, Photo: photo, ID: id})
		}
	})

	// Games (use the same .day-wrapper parser)
	td.Games = ScrapeGames(html, "AGENDADO")

	return td
}

// ---- Club Teams scraping (improved) ----

// ScrapeClubTeams parses /equipas/clube_{id}/ to list all teams of a club.
func ScrapeClubTeams(html string) []models.Team {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil
	}

	var teams []models.Team
	doc.Find(".equipa-item, .team-item, a[href*='/equipa/']").Each(func(_ int, el *goquery.Selection) {
		name := strings.TrimSpace(el.Find(".equipa-nome, .team-name").First().Text())
		if name == "" {
			name = strings.TrimSpace(el.Text())
		}
		href, _ := el.Attr("href")
		if href == "" {
			href, _ = el.Find("a").Attr("href")
		}
		id := strings.TrimPrefix(href, "/equipa/")
		id = strings.TrimSuffix(id, "/")
		logo, _ := el.Find("img").First().Attr("src")
		abrev := strings.TrimSpace(el.Find(".equipa-abreviatura, .abrev").First().Text())

		if name != "" && len(name) > 2 {
			teams = append(teams, models.Team{
				Name: name, ID: id, Logo: logo, Abbreviation: abrev,
			})
		}
	})
	return teams
}

// ---- TugaBasket scraping ----

// TugaBasketStanding holds a TugaBasket standings row.
type TugaBasketStanding struct {
	Group       string `json:"grupo"`
	Team        string `json:"equipa"`
	Position    int    `json:"posicao"`
	Games       int    `json:"jogos"`
	Wins        int    `json:"vitorias"`
	Losses      int    `json:"derrotas"`
	Points      int    `json:"pontos"`
	IsFinished  bool   `json:"is_finished"`
}

// ScrapeTugaBasketStandings parses the TugaBasket HTML response.
func ScrapeTugaBasketStandings(html string) []TugaBasketStanding {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil
	}

	var standings []TugaBasketStanding

	doc.Find(".accordion").Each(func(_ int, acc *goquery.Selection) {
		groupName := strings.TrimSpace(acc.Find(".accordion-title div").First().Text())
		acc.Find("table.standings tbody tr, table.table-striped tbody tr").Each(func(_ int, row *goquery.Selection) {
			cells := row.Find("td")
			if cells.Length() < 5 {
				return
			}
			posText := strings.TrimSpace(cells.Eq(0).Find("span").First().Text())
			if posText == "" {
				posText = strings.TrimSpace(cells.Eq(0).Text())
			}
			standings = append(standings, TugaBasketStanding{
				Group:    groupName,
				Team:     strings.TrimSpace(cells.Eq(1).Text()),
				Position: atoi(posText),
				Games:    atoi(strings.TrimSpace(cells.Eq(2).Text())),
				Wins:     atoi(strings.TrimSpace(cells.Eq(3).Text())),
				Losses:   atoi(strings.TrimSpace(cells.Eq(4).Text())),
				Points:   atoi(strings.TrimSpace(cells.Eq(5).Text())),
			})
		})
	})

	return standings
}
