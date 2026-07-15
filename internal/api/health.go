package api

import (
	"encoding/json"
	"net/http"
)

// Version is set at build time via ldflags.
var Version = "0.1.0"

// HealthResponse is the JSON payload for GET /health.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// Health returns the server health status.
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(HealthResponse{
		Status:  "ok",
		Version: Version,
	})
}
