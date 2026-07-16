package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"github.com/mefrraz/bounce/internal/metrics"
	"runtime"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	apihandler "github.com/mefrraz/bounce/internal/api"
	"github.com/mefrraz/bounce/internal/cache"
	"github.com/mefrraz/bounce/internal/fpbapi"
	"github.com/mefrraz/bounce/internal/httpclient"
	"github.com/mefrraz/bounce/internal/models"
	"github.com/mefrraz/bounce/internal/scheduler"
	"github.com/mefrraz/bounce/internal/ws"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	port := os.Getenv("BOUNCE_PORT")
	if port == "" { port = "3001" }
	dataDir := os.Getenv("BOUNCE_DATA_DIR")
	if dataDir == "" { dataDir = "/data" }
	if err := os.MkdirAll(dataDir, 0755); err != nil { log.Fatalf("data dir: %v", err) }

	store, err := cache.NewStore(filepath.Join(dataDir, "bounce.db"))
	if err != nil { log.Fatalf("cache: %v", err) }
	defer store.Close()

	client := httpclient.New()
	defer client.Stop()

	fpb := fpbapi.New(client, store)
	hub := ws.NewHub(nil, nil)

	sched := scheduler.New(
		func(id string) (*models.Game, error) { d, e := fpb.GetGame(id); if e != nil { return nil, e }; return &d.Game, nil },
		func() ([]models.Game, error) { comps, _ := fpb.GetCompetitions(); var t []models.Game; for _, c := range comps { g, _ := fpb.GetGamesByCompetition(c.ID, cache.CurrentSeason()); for _, gm := range g { if cache.IsToday(gm.Date) { t = append(t, gm) } } }; slog.Info("daily refresh", "games_today", len(t)); return t, nil },
		func(g models.Game) {
			et := "score_update"
			if g.Status == "FINALIZADO" { et = "game_finished" }
			hub.Broadcast(g.ID, ws.Event{Type: et, Data: g})
		},
	)
	hub.SetCallbacks(
		func(id string) { sched.ScheduleGameNow(id) },
		func(id string) { sched.UnscheduleGame(id) },
	)

	corsOrigin := os.Getenv("BOUNCE_CORS_ORIGIN")
	if corsOrigin == "" { corsOrigin = "*" }
	logLevel := os.Getenv("BOUNCE_LOG_LEVEL")
	if logLevel == "" { logLevel = "warn" }
	if logLevel == "debug" {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	}

r := chi.NewRouter()
r.Use(middleware.Logger, middleware.Recoverer, middleware.RealIP, middleware.Compress(5))
r.Use(cors.Handler(cors.Options{AllowedOrigins: []string{corsOrigin}, AllowedMethods: []string{"GET", "POST", "OPTIONS"}, AllowedHeaders: []string{"Content-Type", "Authorization"}, AllowCredentials: false, MaxAge: 86400}))

rl := newRateLimiter(100, time.Minute)
r.Use(rl.middleware)

r.Get("/test", apihandler.TestPage)
r.Get("/health", apihandler.Health)
r.Get("/metrics", metricsHandler)
r.Get("/docs/swagger.json", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "docs/swagger.json") })
r.Get("/docs", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "docs/index.html") })
r.Get("/docs/", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "docs/index.html") })

	apihandler.NewHandler(fpb).RegisterRoutes(r)
	hub.RegisterRoutes(r)
	apihandler.NewInsightsHandler().RegisterRoutes(r)

	sched.Start()
	go func() { fpb.GetCompetitions(); fpb.GetStandings("10902"); slog.Info("pre-warm complete") }()
	srv := &http.Server{Addr: ":" + port, Handler: r}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("starting", "version", apihandler.Version, "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed { log.Fatalf("server: %v", err) }
	}()

	<-ctx.Done()
	slog.Info("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
}

var startTime = time.Now()
var apiKey = ""

func init() { startTime = time.Now(); apiKey = os.Getenv("BOUNCE_API_KEY") }

func metricsHandler(w http.ResponseWriter, _ *http.Request) {
	uptime := int(time.Since(startTime).Seconds())
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "# HELP bounce_uptime_seconds Uptime in seconds\n# TYPE bounce_uptime_seconds gauge\nbounce_uptime_seconds %d\n", uptime)
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Fprintf(w, "# HELP bounce_goroutines Number of goroutines\n# TYPE bounce_goroutines gauge\nbounce_goroutines %d\n", runtime.NumGoroutine())
	fmt.Fprintf(w, "# HELP bounce_memory_bytes Allocated memory\n# TYPE bounce_memory_bytes gauge\nbounce_memory_bytes %d\n", m.Alloc)
	fmt.Fprintf(w, "# HELP bounce_requests_total Total HTTP requests\n# TYPE bounce_requests_total counter\nbounce_requests_total %d\n", metrics.RequestsTotal)
	fmt.Fprintf(w, "# HELP bounce_cache_hits_total Cache hits\n# TYPE bounce_cache_hits_total counter\nbounce_cache_hits_total %d\n", metrics.CacheHitsTotal)
	fmt.Fprintf(w, "# HELP bounce_cache_misses_total Cache misses\n# TYPE bounce_cache_misses_total counter\nbounce_cache_misses_total %d\n", metrics.CacheMissesTotal)
	fmt.Fprintf(w, "# HELP bounce_fpb_requests_total Requests to FPB\n# TYPE bounce_fpb_requests_total counter\nbounce_fpb_requests_total %d\n", metrics.FPBRequestsTotal)
	fmt.Fprintf(w, "# HELP bounce_rate_limited_total Rate-limited requests\n# TYPE bounce_rate_limited_total counter\nbounce_rate_limited_total %d\n", metrics.RateLimitedTotal)
}
