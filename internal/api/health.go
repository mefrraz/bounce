package api

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"strings"
)

//go:embed VERSION
var versionRaw string

var Version = strings.TrimSpace(versionRaw)

func init() {
	if Version == "" { Version = "dev" }
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(HealthResponse{
		Status:  "ok",
		Version: Version,
	})
}
