package api

import (
	_ "embed"
	"net/http"
)

//go:embed test.html
var testHTML []byte

func TestPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Write(testHTML)
}
