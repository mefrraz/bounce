package metrics

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type historyPoint struct {
	Time  string `json:"time"`
	Value uint64 `json:"value"`
}

func HistoryHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	metric := q.Get("metric")
	sinceStr := q.Get("since")
	if sinceStr == "" { sinceStr = "1h" }

	d, _ := time.ParseDuration(sinceStr)
	if d == 0 { d = 1 * time.Hour }

	snapshots := GetHistory(d)
	if len(snapshots) < 2 {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	var points []historyPoint
	prev := snapshots[0]
	for _, s := range snapshots[1:] {
		var v uint64
		switch metric {
		case "requests":
			v = s.Requests - prev.Requests
		case "cache_hits":
			v = s.CacheHits - prev.CacheHits
		case "cache_misses":
			v = s.CacheMisses - prev.CacheMisses
		case "fpb":
			v = s.FPBRequests - prev.FPBRequests
		case "rate_limited":
			v = s.RateLimited - prev.RateLimited
		default:
			v = s.Requests - prev.Requests
		}
		points = append(points, historyPoint{
			Time: s.Time.Format("15:04"), Value: v,
		})
		prev = s
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(points)
}

func HistoryHandlerSimple(w http.ResponseWriter, r *http.Request) {
	minutesStr := r.URL.Query().Get("minutes")
	if minutesStr == "" { minutesStr = "60" }
	minutes, _ := strconv.Atoi(minutesStr)
	since := time.Duration(minutes) * time.Minute

	snapshots := GetHistory(since)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshots)
}
