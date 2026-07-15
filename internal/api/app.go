package api

import (
	_ "embed"
	"net/http"
)

//go:embed app.html
var appHTML []byte

func AppPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(appHTML)
}
