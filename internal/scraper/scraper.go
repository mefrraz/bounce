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
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil { return nil }

	var standings []models.Standing
	doc.Find(".team-row").Each(func(_ int, row *goquery.Selection) {
		// Position: .number h5
		posText := strings.TrimSpace(row.Find(".number h5").First().Text())
		if posText == "" { return }
		pos := atoi(posText)
		if pos == 0 { return }

		// Team name: a[href*='/equipa/'] inside .team-wrapper
		name := strings.TrimSpace(row.Find("a[href*='/equipa/']").First().Text())
		if name == "" { return }

		// Logo: .image-container img
		logo, _ := row.Find(".image-container img").Attr("src")

		// Stats: all h5 elements in order (J, V, D, FC, PM, PS, DIF, PTS)
		var stats []string
		row.Find("h5").Each(func(_ int, h5 *goquery.Selection) {
			stats = append(stats, strings.TrimSpace(h5.Text()))
		})

		s := models.Standing{Position: pos, Team: name, Logo: logo}
		for _, st := range stats {
			switch {
			case st == "J": case st == "V": case st == "D": case st == "FC":
			case st == "PM": case st == "PS": case st == "DIF": case st == "PTS":
				// skip headers
			default:
				switch len(standings) {
				case 0: // already set position
				}
			}
		}
		// Stats order: h5[0]=pos(n), h5[1]=name, then J,V,D,FC,PM,PS,DIF,PTS
		h5s := []string{}
		row.Find("h5").Each(func(_ int, h5 *goquery.Selection) {
			t := strings.TrimSpace(h5.Text())
			if t != "J" && t != "V" && t != "D" && t != "FC" && t != "PM" && t != "PS" && t != "DIF" && t != "PTS" {
				h5s = append(h5s, t)
			}
		})
		// Position is first h5 in .number, name is in link, remaining h5s are stats
		// After the position h5, remaining order: J, V, D, FC, PM, PS, DIF, PTS
		statsOnly := []string{}
		row.Find(".col-2 h5, .col-1 h5, .col-md-1 h5, .col-2.col-md-1 h5").Each(func(_ int, h5 *goquery.Selection) {
			statsOnly = append(statsOnly, strings.TrimSpace(h5.Text()))
		})
		if len(statsOnly) >= 8 {
			s.Played = atoi(statsOnly[0]); s.Won = atoi(statsOnly[1]); s.Lost = atoi(statsOnly[2])
			s.Points = atoi(statsOnly[7])
			v := atoi(statsOnly[4]); s.PointsFor = &v
			v2 := atoi(statsOnly[5]); s.PointsAgainst = &v2
		}
		_ = h5s // keep for reference

		standings = append(standings, s)
	})

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

	// Phase
	detail.Phase = strings.TrimSpace(doc.Find(".phase").First().Text())

	// Teams + abbreviations
	detail.HomeTeam = strings.TrimSpace(doc.Find(".team.home .bigName").First().Text())
	detail.AwayTeam = strings.TrimSpace(doc.Find(".team.away .bigName").First().Text())
	detail.HomeAbbrev = strings.TrimSpace(doc.Find(".team.home .smallName").First().Text())
	detail.AwayAbbrev = strings.TrimSpace(doc.Find(".team.away .smallName").First().Text())

	// Logos
	detail.HomeLogo, _ = doc.Find(".team.home img").First().Attr("src")
	detail.AwayLogo, _ = doc.Find(".team.away img").First().Attr("src")

	// Scores
	pointsSpans := doc.Find(".points span")
	if pointsSpans.Length() >= 2 {
		s1 := atoi(strings.TrimSpace(pointsSpans.Eq(0).Text()))
		s2 := atoi(strings.TrimSpace(pointsSpans.Eq(1).Text()))
		if s1 > 0 || s2 > 0 {
			detail.HomeScore = &s1; detail.AwayScore = &s2
			detail.Status = "FINALIZADO"
		}
	}

	// Date, venue, time
	detail.Date = ParseDatePt(strings.TrimSpace(doc.Find(".date").First().Text()))
	detail.Venue = strings.TrimSpace(doc.Find(".location a").First().Text())
	if href, ok := doc.Find(".location a").First().Attr("href"); ok {
		if m := regexp.MustCompile(`/recinto/(\d+)`).FindStringSubmatch(href); len(m) > 1 {
			detail.PavilionID = m[1]
		}
	}
	timeText := strings.TrimSpace(doc.Find(".match-time").First().Text())
	detail.Time = strings.ReplaceAll(strings.ReplaceAll(timeText, " H ", ":"), " ", "")

	// Attendance
	attText := strings.TrimSpace(doc.Find(".attendance").First().Text())
	if m := regexp.MustCompile(`(\d+)`).FindStringSubmatch(attText); len(m) > 1 {
		detail.Attendance = atoi(m[1])
	}

	// Periods
	doc.Find(".match-period").Each(func(_ int, period *goquery.Selection) {
		scoreText := strings.TrimSpace(period.Find(".partial-score").First().Text())
		parts := strings.Split(scoreText, "-")
		if len(parts) == 2 {
			detail.Periods = append(detail.Periods, models.Period{
				Number: len(detail.Periods) + 1,
				HomeScore: atoi(strings.TrimSpace(parts[0])),
				AwayScore: atoi(strings.TrimSpace(parts[1])),
			})
		}
	})

	// Game Leaders (top performers) with hardcoded category fallback
	categories := []string{"PONTOS", "RESSALTOS", "ASSISTÊNCIAS", "ROUBOS", "DESARMES"}
	doc.Find(".performance-wrapper").Each(func(idx int, w *goquery.Selection) {
		cat := strings.TrimSpace(w.Find(".category-name").First().Text())
		if cat == "" { cat = strings.TrimSpace(w.Find(".divider span").First().Text()) }
		if cat == "" || cat == "-" {
			if idx < len(categories) { cat = categories[idx] } else { cat = "" }
		}
		players := w.Find(".player")
		if players.Length() >= 2 {
			// Try .valor first, then .divider children
			homeStat := strings.TrimSpace(players.Eq(0).Find(".valor").First().Text())
			awayStat := strings.TrimSpace(players.Eq(1).Find(".valor").First().Text())
			if homeStat == "" || awayStat == "" {
				divider := w.Find(".divider")
				if dividerChildren := divider.Children(); dividerChildren.Length() >= 3 {
					if homeStat == "" { homeStat = strings.TrimSpace(dividerChildren.Eq(0).Text()) }
					if awayStat == "" { awayStat = strings.TrimSpace(dividerChildren.Eq(2).Text()) }
				}
			}
			homePhoto, _ := players.Eq(0).Find("img").First().Attr("src")
			awayPhoto, _ := players.Eq(1).Find("img").First().Attr("src")
			detail.GameLeaders = append(detail.GameLeaders, models.GameLeader{
				Category: cat,
				Home: models.LeaderPlayer{
					Name:  strings.TrimSpace(players.Eq(0).Find(".name").First().Text()),
					Stat:  homeStat,
					Photo: homePhoto,
				},
				Away: models.LeaderPlayer{
					Name:  strings.TrimSpace(players.Eq(1).Find(".name").First().Text()),
					Stat:  awayStat,
					Photo: awayPhoto,
				},
			})
		}
	})

	// Full box score (22 fields)
	parseBoxScore := func(sel *goquery.Selection) []models.PlayerStat {
		var stats []models.PlayerStat
		sel.Find("tbody tr").Each(func(_ int, row *goquery.Selection) {
			cells := row.Find("td")
			if cells.Length() < 3 { return }
			name := strings.TrimSpace(cells.Eq(1).Text())
			if name == "" || name == "Total" { return }
			s := models.PlayerStat{
				Name: name, Number: atoi(strings.TrimSpace(cells.Eq(0).Text())),
				PTS: atoi(strings.TrimSpace(cells.Eq(2).Text())),
				Photo: func() string { p, _ := cells.Eq(1).Find("img").First().Attr("src"); return p }(),
			}
			if cells.Length() > 3 { s.MIN = strings.TrimSpace(cells.Eq(3).Text()) }
			if cells.Length() > 4 { s.L2 = strings.TrimSpace(cells.Eq(4).Text()) }
			if cells.Length() > 5 { s.L2Pct = strings.TrimSpace(cells.Eq(5).Text()) }
			if cells.Length() > 6 { s.L3 = strings.TrimSpace(cells.Eq(6).Text()) }
			if cells.Length() > 7 { s.L3Pct = strings.TrimSpace(cells.Eq(7).Text()) }
			if cells.Length() > 8 { s.LL = strings.TrimSpace(cells.Eq(8).Text()) }
			if cells.Length() > 9 { s.LLPct = strings.TrimSpace(cells.Eq(9).Text()) }
			if cells.Length() > 10 { s.RO = atoi(strings.TrimSpace(cells.Eq(10).Text())) }
			if cells.Length() > 11 { s.RD = atoi(strings.TrimSpace(cells.Eq(11).Text())) }
			if cells.Length() > 12 { s.RT = atoi(strings.TrimSpace(cells.Eq(12).Text())) }
			if cells.Length() > 13 { s.AS = atoi(strings.TrimSpace(cells.Eq(13).Text())) }
			if cells.Length() > 14 { s.RB = atoi(strings.TrimSpace(cells.Eq(14).Text())) }
			if cells.Length() > 15 { s.TO = atoi(strings.TrimSpace(cells.Eq(15).Text())) }
			if cells.Length() > 16 { s.DL = atoi(strings.TrimSpace(cells.Eq(16).Text())) }
			if cells.Length() > 17 { s.FC = atoi(strings.TrimSpace(cells.Eq(17).Text())) }
			stats = append(stats, s)
		})
		return stats
	}
	doc.Find(".game-detail .table-responsive, table.ficha-tabela").Each(func(i int, table *goquery.Selection) {
		s := parseBoxScore(table)
		if i == 0 && len(s) > 0 { detail.HomeStats = s }
		if i == 1 && len(s) > 0 { detail.AwayStats = s }
	})

	// Top Performers (Duelo card): .players-wrapper
	if topWrapper := doc.Find(".players-wrapper").First(); topWrapper.Length() > 0 {
		players := topWrapper.Find(".player")
		if players.Length() >= 2 {
			detail.TopPerfCasa = models.TopPerformer{
				Name: strings.TrimSpace(players.Eq(0).Find(".name").First().Text()),
			}
			detail.TopPerfCasa.Photo, _ = players.Eq(0).Find("img").First().Attr("src")
			detail.TopPerfFora = models.TopPerformer{
				Name: strings.TrimSpace(players.Eq(players.Length()-1).Find(".name").First().Text()),
			}
			detail.TopPerfFora.Photo, _ = players.Eq(players.Length()-1).Find("img").First().Attr("src")
		}
	}
	// Top Performer Stats: .topPerformers-stats .type-key-stats.big
	doc.Find(".topPerformers-stats .type-key-stats.big").Each(func(_ int, el *goquery.Selection) {
		label := strings.TrimSpace(el.Find(".double p").First().Text())
		vals := el.Find(".one-line-graph.single p")
		cs := strings.TrimSpace(vals.Eq(0).Text())
		fs := strings.TrimSpace(vals.Eq(1).Text())
		if label != "" && (cs != "" || fs != "") {
			detail.TopPerfStats = append(detail.TopPerfStats, models.TopPerfStat{Label: label, Casa: cs, Fora: fs})
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
	td := &TeamDetail{}

	// Dribly: team name from <div class="team-nome">NAME</div>
	if nm := regexp.MustCompile(`<div class="team-nome">\s*([^<]+)\s*</div>`).FindStringSubmatch(html); len(nm) > 1 {
		td.Name = strings.TrimSpace(nm[1])
	}

	// Photo: <div class="team-right"> <img src="..." />
	if pm := regexp.MustCompile(`<div class="team-right[^"]*">\s*<img\s+src="([^"]+)"`).FindStringSubmatch(html); len(pm) > 1 && !strings.Contains(pm[1], "noplayer") {
		td.Photo = pm[1]
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err == nil {
		doc.Find(".player").Each(func(_ int, el *goquery.Selection) {
			photo, _ := el.Find("img").First().Attr("src")
			name := strings.TrimSpace(el.Find(".info").First().Text())
			if name == "" { name = strings.TrimSpace(el.Text()) }
			if len(name) > 2 && len(name) < 40 {
				td.Players = append(td.Players, TeamPlayer{Name: name, Photo: photo})
			}
		})
	}

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

// ---- TugaBasket Player Stats ----

type TBPlayerStat struct {
	Name  string `json:"nome"`
	Team  string `json:"equipa"`
	Games int    `json:"jogos"`
	MIN   string `json:"min"`
	PTS   string `json:"pts"`
	L2Pct string `json:"l2pct"`
	L3Pct string `json:"l3pct"`
	LLPct string `json:"llpct"`
	RD    string `json:"rd"`
	RO    string `json:"ro"`
	TR    string `json:"tr"`
	AS    string `json:"as"`
	RB    string `json:"rb"`
	TO    string `json:"to"`
	DL    string `json:"dl"`
	FC    string `json:"fc"`
	FS    string `json:"fs"`
	VAL   string `json:"val"`
}

func ScrapeTugaBasketPlayers(html string) []TBPlayerStat {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil { return nil }
	var players []TBPlayerStat
	doc.Find("table tbody tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 5 { return }
		p := TBPlayerStat{}
		p.Name = strings.TrimSpace(cells.Eq(0).Text())
		p.Team = strings.TrimSpace(cells.Eq(1).Text())
		p.Games = atoi(strings.TrimSpace(cells.Eq(2).Text()))
		if cells.Length() > 3 { p.MIN = strings.TrimSpace(cells.Eq(3).Text()) }
		if cells.Length() > 4 { p.PTS = strings.TrimSpace(cells.Eq(4).Text()) }
		if cells.Length() > 5 { p.L2Pct = strings.TrimSpace(cells.Eq(5).Text()) }
		if cells.Length() > 6 { p.L3Pct = strings.TrimSpace(cells.Eq(6).Text()) }
		if cells.Length() > 7 { p.LLPct = strings.TrimSpace(cells.Eq(7).Text()) }
		if cells.Length() > 8 { p.RD = strings.TrimSpace(cells.Eq(8).Text()) }
		if cells.Length() > 9 { p.RO = strings.TrimSpace(cells.Eq(9).Text()) }
		if cells.Length() > 10 { p.TR = strings.TrimSpace(cells.Eq(10).Text()) }
		if cells.Length() > 11 { p.AS = strings.TrimSpace(cells.Eq(11).Text()) }
		if cells.Length() > 12 { p.RB = strings.TrimSpace(cells.Eq(12).Text()) }
		if cells.Length() > 13 { p.TO = strings.TrimSpace(cells.Eq(13).Text()) }
		if cells.Length() > 14 { p.DL = strings.TrimSpace(cells.Eq(14).Text()) }
		if cells.Length() > 15 { p.FC = strings.TrimSpace(cells.Eq(15).Text()) }
		if cells.Length() > 16 { p.FS = strings.TrimSpace(cells.Eq(16).Text()) }
		if cells.Length() > 17 { p.VAL = strings.TrimSpace(cells.Eq(17).Text()) }
		if p.Name != "" { players = append(players, p) }
	})
	return players
}

// ---- TugaBasket Team Stats ----

type TBTeamStat struct {
	Name  string `json:"nome"`
	Games int    `json:"jogos"`
	PTS   string `json:"pts"`
	L2Pct string `json:"l2pct"`
	L3Pct string `json:"l3pct"`
	LLPct string `json:"llpct"`
	RD    string `json:"rd"`
	RO    string `json:"ro"`
	TR    string `json:"tr"`
	AS    string `json:"as"`
}

func ScrapeTugaBasketTeams(html string) []TBTeamStat {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil { return nil }
	var teams []TBTeamStat
	doc.Find("table tbody tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 3 { return }
		t := TBTeamStat{}
		t.Name = strings.TrimSpace(cells.Eq(0).Text())
		t.Games = atoi(strings.TrimSpace(cells.Eq(1).Text()))
		if cells.Length() > 2 { t.PTS = strings.TrimSpace(cells.Eq(2).Text()) }
		if cells.Length() > 3 { t.L2Pct = strings.TrimSpace(cells.Eq(3).Text()) }
		if cells.Length() > 4 { t.L3Pct = strings.TrimSpace(cells.Eq(4).Text()) }
		if cells.Length() > 5 { t.LLPct = strings.TrimSpace(cells.Eq(5).Text()) }
		if cells.Length() > 6 { t.RD = strings.TrimSpace(cells.Eq(6).Text()) }
		if cells.Length() > 7 { t.RO = strings.TrimSpace(cells.Eq(7).Text()) }
		if cells.Length() > 8 { t.TR = strings.TrimSpace(cells.Eq(8).Text()) }
		if cells.Length() > 9 { t.AS = strings.TrimSpace(cells.Eq(9).Text()) }
		if t.Name != "" { teams = append(teams, t) }
	})
	return teams
}

// ── Competition Player Stats ──


