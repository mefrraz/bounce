package clubs

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Club represents a basketball club as stored in the JSON database.
type Club struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	Slug          string  `json:"slug,omitempty"`
	ShortName     string  `json:"short_name"`
	SearchName    string  `json:"search_name,omitempty"`
	LogoURL       string  `json:"logo_url,omitempty"`
	PrimaryColor  string  `json:"primary_color,omitempty"`
	LogoSecondary string  `json:"logo_secondary,omitempty"`
	Priority      int     `json:"priority,omitempty"`
	EloRating     float64 `json:"elo_rating,omitempty"`
	LogoPattern   string  `json:"-"`
}

const fpbClubesURL = "https://www.fpb.pt/clubes/"
const clubsFile = "clubs.json"

var clubsMu sync.RWMutex
var dataDir string
var clubsData []Club
var byLogo map[string]*Club

// Init loads clubs from disk and sets up the data directory.
func Init(dir string) error {
	dataDir = dir
	return loadFromDisk()
}

func loadFromDisk() error {
	path := filepath.Join(dataDir, clubsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	clubsMu.Lock()
	defer clubsMu.Unlock()
	if err := json.Unmarshal(data, &clubsData); err != nil {
		return err
	}
	rebuildIndex()
	return nil
}

func rebuildIndex() {
	byLogo = make(map[string]*Club)
	for i := range clubsData {
		c := &clubsData[i]
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

func saveToDisk() error {
	clubsMu.RLock()
	defer clubsMu.RUnlock()
	return saveToDiskLocked()
}

func saveToDiskLocked() error {
	data, _ := json.MarshalIndent(clubsData, "", "  ")
	path := filepath.Join(dataDir, clubsFile)
	return os.WriteFile(path, data, 0644)
}

// RefreshFromFPB scrapes fpb.pt/clubes, extracts club data, and merges with existing.
// Returns counts: updated, added, errors.
func RefreshFromFPB() (updated int, added int, errs int) {
	log.Printf("[clubs] scraping %s", fpbClubesURL)
	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequest("GET", fpbClubesURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Bounce/1.0)")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[clubs] fetch error: %v", err)
		return 0, 0, 1
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	scraped := scrapeClubsHTML(html)
	log.Printf("[clubs] scraped %d clubs from FPB", len(scraped))

	if len(scraped) == 0 {
		return 0, 0, 1
	}

	// Merge with existing
	clubsMu.Lock()
	defer clubsMu.Unlock()

	existing := make(map[int]*Club)
	for i := range clubsData {
		existing[clubsData[i].ID] = &clubsData[i]
	}

	for _, sc := range scraped {
		if ex, ok := existing[sc.ID]; ok {
			// Update scrapeable fields, preserve manual edits
			if ex.PrimaryColor == "" || ex.PrimaryColor == "#7C3AED" || ex.PrimaryColor == "#000000" {
				ex.PrimaryColor = sc.PrimaryColor
			}
			if ex.Name != sc.Name {
				ex.Name = sc.Name
			}
			ex.LogoURL = sc.LogoURL
			// Preserve existing short_name — FPB scraper can't extract it
			updated++
		} else {
			// New club
			newClub := Club{
				ID:           sc.ID,
				Name:         sc.Name,
				ShortName:    sc.ShortName,
				PrimaryColor: sc.PrimaryColor,
				LogoURL:      sc.LogoURL,
				EloRating:    1500,
				SearchName: strings.ToLower(strings.Map(func(r rune) rune {
					if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
						return r
					}
					return -1
				}, strings.ToLower(sc.Name))),
				Slug: slugify(sc.Name),
			}
			clubsData = append(clubsData, newClub)
			existing[sc.ID] = &clubsData[len(clubsData)-1]
			added++
		}
	}

	rebuildIndex()
	if err := saveToDiskLocked(); err != nil {
		log.Printf("[clubs] save error: %v", err)
		errs++
	}

	log.Printf("[clubs] refresh done: %d updated, %d added, %d errors", updated, added, errs)
	return
}

type scrapedClub struct {
	ID           int
	Name         string
	ShortName    string
	PrimaryColor string
	LogoURL      string
}

func scrapeClubsHTML(html string) []scrapedClub {
	var clubs []scrapedClub

	// Find all .clube divs
	clubRe := regexp.MustCompile(`<div class="clube"[^>]*>[\s\S]*?<a href="([^"]*calendario/clube_(\d+)[^"]*)"[^>]*>[\s\S]*?<div class="clube-body"[^>]*style="[^"]*background-color:\s*(#[0-9a-fA-F]+)[^"]*"[\s\S]*?<img[^>]*src="([^"]*)"[^>]*>[\s\S]*?<div class="clube-shortname">\s*([^<]*)\s*</div>[\s\S]*?<div class="clube-name">\s*([^<]*)\s*</div>`)
	matches := clubRe.FindAllStringSubmatch(html, -1)

	for _, m := range matches {
		id, _ := strconv.Atoi(m[2])
		clubs = append(clubs, scrapedClub{
			ID:           id,
			Name:         strings.TrimSpace(m[6]),
			ShortName:    strings.TrimSpace(m[5]),
			PrimaryColor: m[3],
			LogoURL:      m[4],
		})
	}

	// Fallback: simpler regex if the main one fails
	if len(clubs) == 0 {
		fallbackRe := regexp.MustCompile(`calendario/clube_(\d+)[^"]*"[^>]*>[\s\S]*?background-color:\s*(#[0-9a-fA-F]+)[\s\S]*?src="([^"]*)"[\s\S]*?clube-name">\s*([^<]+)`)
		fmatches := fallbackRe.FindAllStringSubmatch(html, -1)
		for _, m := range fmatches {
			id, _ := strconv.Atoi(m[1])
			clubs = append(clubs, scrapedClub{
				ID:           id,
				Name:         strings.TrimSpace(m[4]),
				PrimaryColor: m[2],
				LogoURL:      m[3],
			})
		}
	}

	return clubs
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' || r == '-' {
			return r
		}
		return -1
	}, s)
	s = strings.TrimSpace(s)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// UpdateClub updates a single club's editable fields and saves to disk.
func UpdateClub(id int, name, shortName, primaryColor, logoURL string, priority int) error {
	clubsMu.Lock()
	defer clubsMu.Unlock()
	for i := range clubsData {
		if clubsData[i].ID == id {
			if name != "" {
				clubsData[i].Name = name
			}
			if shortName != "" {
				clubsData[i].ShortName = shortName
			}
			if primaryColor != "" {
				clubsData[i].PrimaryColor = primaryColor
			}
			if logoURL != "" {
				clubsData[i].LogoURL = logoURL
			}
			clubsData[i].Priority = priority
			rebuildIndex()
			return saveToDiskLocked()
		}
	}
	return fmt.Errorf("club %d not found", id)
}

// All returns a copy of the current clubs slice.
func All() []Club {
	clubsMu.RLock()
	defer clubsMu.RUnlock()
	result := make([]Club, len(clubsData))
	copy(result, clubsData)
	return result
}

// MatchByLogo finds a club whose logo URL pattern appears in the given URL.
func MatchByLogo(logoURL string) *Club {
	if logoURL == "" {
		return nil
	}
	clubsMu.RLock()
	defer clubsMu.RUnlock()
	parts := strings.Split(logoURL, "/")
	last := parts[len(parts)-1]
	if dot := strings.LastIndex(last, "."); dot > 0 {
		last = last[:dot]
	}
	pattern := strings.ToLower(last)
	if c, ok := byLogo[pattern]; ok {
		return c
	}
	for _, c := range clubsData {
		if c.LogoPattern != "" && strings.Contains(strings.ToLower(logoURL), c.LogoPattern) {
			return &c
		}
	}
	return nil
}

// NormalizeTeam returns canonical short_name and logo for a team.
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

// StartDailyRefresh runs a background goroutine that refreshes clubs from FPB every 24h.
func StartDailyRefresh() {
	go func() {
		for {
			time.Sleep(24 * time.Hour)
			log.Printf("[clubs] daily refresh starting")
			RefreshFromFPB()
		}
	}()
}