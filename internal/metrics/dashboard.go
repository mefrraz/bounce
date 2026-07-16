package metrics

import (
	_ "embed"
	"net/http"
)

//go:embed dashboard.html
var dashboardHTML []byte

func DashboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(dashboardHTML)
}
