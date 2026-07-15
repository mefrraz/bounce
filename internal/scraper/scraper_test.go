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
			t.Errorf("ParseDatePt(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestScrapeStandingsEmpty(t *testing.T) {
	if len(ScrapeStandings("")) != 0 {
		t.Error("expected empty")
	}
}

func TestScrapeGamesEmpty(t *testing.T) {
	if len(ScrapeGames("", "AGENDADO")) != 0 {
		t.Error("expected empty")
	}
}

func TestExtractFaseIDs(t *testing.T) {
	html := `<li class="option" tag="Fase Regular" value="30969"></li>`
	fases := ExtractFaseIDs(html)
	if len(fases) != 1 || fases[0].ID != "30969" {
		t.Errorf("got %v", fases)
	}
}
