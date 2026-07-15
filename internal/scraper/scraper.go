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

			// Team names: Dribly uses .team-container .fullName or .sigla
			containers := el.Find(".team-container")
			homeName := ""
			awayName := ""
			if containers.Length() >= 2 {
				homeName = strings.TrimSpace(containers.Eq(0).Find(".fullName").First().Text())
				if homeName == "" {
					homeName = strings.TrimSpace(containers.Eq(0).Find(".sigla").First().Text())
				}
				awayName = strings.TrimSpace(containers.Eq(1).Find(".fullName").First().Text())
				if awayName == "" {
					awayName = strings.TrimSpace(containers.Eq(1).Find(".sigla").First().Text())
				}
			}
			if homeName == "" && awayName == "" {
				return
			}

			// Logos: Dribly uses .team-container .image-container img
			homeLogo, _ := containers.Eq(0).Find(".image-container img").Attr("src")
			awayLogo, _ := containers.Eq(1).Find(".image-container img").Attr("src")

			// Scores: Dribly uses .results_wrapper h3.results_text
			var hs, as *int
			status := defaultStatus
			resultsWrapper := el.Find(".results_wrapper")
			if resultsWrapper.Length() > 0 {
				scoreEls := resultsWrapper.Find("h3.results_text")
				if scoreEls.Length() >= 2 {
					status = "FINALIZADO"
					v1 := atoi(strings.TrimSpace(scoreEls.Eq(0).Text()))
					v2 := atoi(strings.TrimSpace(scoreEls.Eq(1).Text()))
					hs = &v1
					as = &v2
				}
			}

			// Time or score fallback: Dribly uses .hour h3
			hourEl := el.Find(".hour h3")
			hourText := strings.TrimSpace(hourEl.Text())
			timeStr := ""
			if status == "AGENDADO" && strings.Contains(hourText, "-") && !strings.Contains(hourText, "H") {
				parts := strings.Split(hourText, "-")
				if len(parts) == 2 {
					s1 := atoi(strings.TrimSpace(parts[0]))
					s2 := atoi(strings.TrimSpace(parts[1]))
					if s1 > 0 || s2 > 0 {
						status = "FINALIZADO"
						hs = &s1
						as = &s2
					}
				}
			}
			if status == "AGENDADO" {
				timeStr = strings.ReplaceAll(hourText, "H", ":")
				timeStr = strings.ReplaceAll(timeStr, " ", "")
			}

			// Venue: Dribly uses .location-wrapper b
			locEl := el.Find(".location-wrapper b")
			venue := strings.TrimSpace(locEl.Text())

			// Competition name
			compEl := el.Find(".competition span")
			compText := strings.TrimSpace(compEl.Text())
			compName := ""
			compCategory := ""
			if strings.Contains(compText, "|") {
				parts := strings.SplitN(compText, "|", 2)
				compCategory = strings.TrimSpace(parts[0])
				compName = strings.TrimSpace(parts[1])
			} else {
				compName = compText
			}

			if id == "" {
				id = isoDate + "-" + slugify(homeName) + "-" + slugify(awayName)
			}

			games = append(games, models.Game{
				ID: id, Date: isoDate, Time: timeStr,
				HomeTeam: homeName, AwayTeam: awayName,
				HomeScore: hs, AwayScore: as,
				Venue: venue, Status: status,
				HomeLogo: homeLogo, AwayLogo: awayLogo,
				Competition: compName, Category: compCategory,
			})
		})
	})
	return games
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	return s
}

func ScrapeGameDetail(html string) (*models.GameDetail, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	detail := &models.GameDetail{}

	// Teams: .team.home .bigName / .team.away .bigName (Dribly selectors)
	detail.HomeTeam = strings.TrimSpace(doc.Find(".team.home .bigName").First().Text())
	detail.AwayTeam = strings.TrimSpace(doc.Find(".team.away .bigName").First().Text())

	// Logos: .team.home img / .team.away img
	detail.HomeLogo, _ = doc.Find(".team.home img").First().Attr("src")
	detail.AwayLogo, _ = doc.Find(".team.away img").First().Attr("src")

	// Scores: .points span (Dribly: 2 spans with scores)
	pointsSpans := doc.Find(".points span")
	if pointsSpans.Length() >= 2 {
		s1 := atoi(strings.TrimSpace(pointsSpans.Eq(0).Text()))
		s2 := atoi(strings.TrimSpace(pointsSpans.Eq(1).Text()))
		if s1 > 0 || s2 > 0 {
			detail.HomeScore = &s1
			detail.AwayScore = &s2
			detail.Status = "FINALIZADO"
		}
	}

	// Date: .date (Dribly: "30 MAI 2026")
	dateText := strings.TrimSpace(doc.Find(".date").First().Text())
	detail.Date = ParseDatePt(dateText)

	// Venue: .location a
	detail.Venue = strings.TrimSpace(doc.Find(".location a").First().Text())

	// Time: .match-time
	timeText := strings.TrimSpace(doc.Find(".match-time").First().Text())
	detail.Time = strings.ReplaceAll(strings.ReplaceAll(timeText, " H ", ":"), " ", "")

	// Periods: .match-period .partial-score (Dribly: "21 - 20")
	doc.Find(".match-period").Each(func(_ int, period *goquery.Selection) {
		label := strings.TrimSpace(period.Find("p").First().Text())
		scoreText := strings.TrimSpace(period.Find(".partial-score").First().Text())
		parts := strings.Split(scoreText, "-")
		if len(parts) == 2 {
			hs := atoi(strings.TrimSpace(parts[0]))
			as := atoi(strings.TrimSpace(parts[1]))
			detail.Periods = append(detail.Periods, models.Period{
				Number: len(detail.Periods) + 1, HomeScore: hs, AwayScore: as,
			})
			_ = label
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

// ---- TugaBasket Game Results ----

type TBGameResult struct {
	Date      string `json:"data"`
	HomeTeam  string `json:"equipa_casa"`
	AwayTeam  string `json:"equipa_fora"`
	Score     string `json:"resultado"`
	Phase     string `json:"fase"`
}

func ScrapeTugaBasketGames(html string) []TBGameResult {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil { return nil }
	var results []TBGameResult
	doc.Find("table tbody tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 5 { return }
		results = append(results, TBGameResult{
			Date: strings.TrimSpace(cells.Eq(1).Text()),
			HomeTeam: strings.TrimSpace(cells.Eq(2).Text()),
			Score: strings.TrimSpace(cells.Eq(3).Text()),
			AwayTeam: strings.TrimSpace(cells.Eq(4).Text()),
			Phase: strings.TrimSpace(cells.Eq(5).Text()),
		})
	})
	return results
}
