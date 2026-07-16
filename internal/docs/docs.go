package docs

import (
	_ "embed"
	"net/http"
)

//go:embed docs.html
var docsHTML []byte

func Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write(docsHTML)
}
