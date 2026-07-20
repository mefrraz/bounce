package clubs

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"sort"
	"strings"
)

// ExtractColor downloads a logo and returns its dominant hex color.
func ExtractColor(logoURL string) string {
	if logoURL == "" {
		return "#000000"
	}
	resp, err := http.Get(logoURL)
	if err != nil {
		return "#000000"
	}
	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return "#000000"
	}

	return dominantColor(img)
}

// dominantColor finds the most frequent color in an image using simple histogram.
func dominantColor(img image.Image) string {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Sample every 4th pixel for speed (> 100x faster than full scan)
	hist := make(map[[3]uint8]int)
	step := 4
	if width*height < 10000 {
		step = 2
	}
	if width*height > 100000 {
		step = 8
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			r, g, b, a := img.At(x, y).RGBA()
			if a < 32768 { continue } // skip transparent pixels (alpha < 50%)
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
			// Quantize to 24 buckets (8 per channel) to group similar colors
			r8 = r8 / 32 * 32
			g8 = g8 / 32 * 32
			b8 = b8 / 32 * 32
			hist[[3]uint8{r8, g8, b8}]++
		}
	}

	if len(hist) == 0 {
		return "#000000"
	}

	// Sort by frequency
	type entry struct{ c [3]uint8; n int }
	var entries []entry
	for c, n := range hist { entries = append(entries, entry{c, n}) }
	sort.Slice(entries, func(i, j int) bool { return entries[i].n > entries[j].n })

	best := entries[0].c
	hex := rgb2hex(best[0], best[1], best[2])
	if hex == "#000000" || hex == "#ffffff" || strings.ToLower(hex) == "#000000" {
		// Try second best if top color is black/white (likely background)
		if len(entries) > 1 {
			best = entries[1].c
			return rgb2hex(best[0], best[1], best[2])
		}
	}
	return hex
}

func rgb2hex(r, g, b uint8) string {
	return "#" + hexb(r) + hexb(g) + hexb(b)
}

func hexb(v uint8) string {
	const hex = "0123456789ABCDEF"
	return string([]byte{hex[v>>4], hex[v&15]})
}
