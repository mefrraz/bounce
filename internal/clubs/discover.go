package clubs

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

const (
	fpbCalendarURL   = "https://www.fpb.pt/calendario/clube_%d/"
	discoveryDelay   = 300 * time.Millisecond // per-request delay
	discoveryWorkers = 5
	discoveryMaxID   = 10000
)

// titleNameRe extracts club name from <title>Calendário de Basquetebol em Portugal - Nome</title>.
var titleNameRe = regexp.MustCompile(`<title>Calendário de Basquetebol em Portugal - (.+?)</title>`)

// ogImageRe extracts logo URL from <meta property="og:image" content="URL" />.
var ogImageRe = regexp.MustCompile(`<meta property="og:image" content="([^"]+)"`)

// DiscoverAllClubs brute-forces club discovery by trying every ID from 1 to maxID
// against fpb.pt/calendario/clube_<id>/. Valid clubs have a club name in the <title>;
// invalid ones show the generic title without a club name.
// Already-known clubs are updated if name or logo changed. Unknown IDs with no
// valid club are skipped, making it idempotent and resumable.
func DiscoverAllClubs(maxID int) {
	if maxID <= 0 {
		maxID = discoveryMaxID
	}
	clubsMu.RLock()
	knownIDs := make(map[int]bool)
	for _, c := range clubsData {
		if c.ID > 0 {
			knownIDs[c.ID] = true
		}
	}
	clubsMu.RUnlock()

	// Build list of IDs to check
	var toCheck []int
	for id := 1; id <= maxID; id++ {
		if !knownIDs[id] {
			toCheck = append(toCheck, id)
		}
	}
	total := len(toCheck)
	if total == 0 {
		log.Printf("[discover] all %d IDs already known, nothing to do", maxID)
		return
	}
	log.Printf("[discover] scanning %d unknown IDs (1-%d), %d already known",
		total, maxID, maxID-total)

	start := time.Now()
	client := &http.Client{Timeout: 20 * time.Second}

	sem := make(chan struct{}, discoveryWorkers)
	results := make(chan *Club, discoveryWorkers*2)
	done := make(chan struct{})

	var newClubs int64
	var updated int64
	var skipped int64

	// Build ID-to-index map for fast upsert
	clubsMu.RLock()
	idToIndex := make(map[int]int)
	for i, c := range clubsData {
		if c.ID > 0 {
			idToIndex[c.ID] = i
		}
	}
	clubsMu.RUnlock()

	// Collector goroutine — upsert: update if exists, append if new
	go func() {
		for c := range results {
			clubsMu.Lock()
			if idx, ok := idToIndex[c.ID]; ok {
				// Update existing — preserve manual fields
				existing := &clubsData[idx]
				if existing.Name != c.Name {
					existing.Name = c.Name
					existing.Slug = c.Slug
					existing.SearchName = c.SearchName
				}
				if c.LogoURL != "" && existing.LogoURL != c.LogoURL {
					existing.LogoURL = c.LogoURL
				}
				// Keep existing short_name if set (may differ from name)
				if existing.ShortName == "" || existing.ShortName == c.Name {
					existing.ShortName = c.Name
				}
				atomic.AddInt64(&updated, 1)
			} else {
				clubsData = append(clubsData, *c)
				idToIndex[c.ID] = len(clubsData) - 1
				atomic.AddInt64(&newClubs, 1)
			}
			nc := atomic.LoadInt64(&newClubs)
			up := atomic.LoadInt64(&updated)
			if (nc+up)%50 == 0 {
				saveToDiskLocked()
			}
			clubsMu.Unlock()
		}
		close(done)
	}()

	// Progress reporter goroutine
	stopProgress := make(chan struct{})
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				nc := atomic.LoadInt64(&newClubs)
				up := atomic.LoadInt64(&updated)
				sk := atomic.LoadInt64(&skipped)
				checked := int(nc + up + sk)
				if checked == 0 {
					continue
				}
				pct := checked * 100 / total
				elapsed := time.Since(start)
				eta := time.Duration(0)
				if checked > 0 {
					eta = elapsed * time.Duration(total-checked) / time.Duration(checked)
				}
				log.Printf("[discover] %3d%% · %d/%d · %d new · %d upd · ETA %v",
					pct, checked, total, nc, up, eta.Round(time.Second))
			case <-stopProgress:
				return
			}
		}
	}()

	// Worker dispatcher
	for _, id := range toCheck {
		sem <- struct{}{}
		go func(clubID int) {
			defer func() { <-sem }()
			time.Sleep(discoveryDelay)

			c, err := fetchClubPage(client, clubID)
			if err != nil {
				atomic.AddInt64(&skipped, 1)
				return
			}
			if c == nil {
				atomic.AddInt64(&skipped, 1)
				return
			}
			results <- c
		}(id)
	}

	// Wait for all workers
	for i := 0; i < discoveryWorkers; i++ {
		sem <- struct{}{}
	}
	close(results)
	<-done
	close(stopProgress)

	// Final save + rebuild index
	clubsMu.Lock()
	rebuildIndex()
	saveToDiskLocked()
	clubsMu.Unlock()

	nc := atomic.LoadInt64(&newClubs)
	up := atomic.LoadInt64(&updated)
	sk := atomic.LoadInt64(&skipped)
	elapsed := time.Since(start).Round(time.Second)
	log.Printf("[discover] done: %d new, %d updated, %d skipped in %v", nc, up, sk, elapsed)
}

// fetchClubPage fetches the calendar page for a club and extracts name + logo.
// Returns nil if the club is invalid (no club name in title).
func fetchClubPage(client *http.Client, id int) (*Club, error) {
	url := fmt.Sprintf(fpbCalendarURL, id)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Bounce/1.0)")
	req.Header.Set("Accept", "text/html")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, nil
	}

	// Read body — limit to 64KB to avoid huge pages
	body, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		return nil, err
	}
	html := string(body)

	// Extract club name from <title>
	m := titleNameRe.FindStringSubmatch(html)
	if m == nil {
		return nil, nil // no club name → invalid
	}
	name := strings.TrimSpace(m[1])
	if name == "" {
		return nil, nil
	}

	// Extract logo from og:image
	var logoURL string
	if lm := ogImageRe.FindStringSubmatch(html); lm != nil {
		logoURL = strings.TrimSpace(lm[1])
	}

	// Skip FPB default image (not a real club logo)
	if strings.Contains(logoURL, "1200x628.png") || strings.Contains(logoURL, "Logo-FPB.jpg") {
		logoURL = ""
	}

	log.Printf("[discover] club: ID=%d name=%q logo=%s", id, name, logoURL)

	return &Club{
		ID:         id,
		Name:       name,
		ShortName:  name,
		LogoURL:    logoURL,
		EloRating:  1500,
		Priority:   4,
		Slug:       slugify(name),
		SearchName: slugifyLower(name),
	}, nil
}
