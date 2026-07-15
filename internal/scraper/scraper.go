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

func ScrapeGameDetail(html string) (*models.GameDetail, error) { return nil, nil }

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
