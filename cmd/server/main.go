package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
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
		func() ([]models.Game, error) { return nil, nil },
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

r := chi.NewRouter()
r.Use(middleware.Logger, middleware.Recoverer, middleware.RealIP, middleware.Compress(5))
r.Use(cors.Handler(cors.Options{AllowedOrigins: []string{"*"}, AllowedMethods: []string{"GET", "POST", "OPTIONS"}, AllowedHeaders: []string{"Content-Type", "Authorization"}, AllowCredentials: false, MaxAge: 86400}))

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
	defer sched.Stop()

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

func init() { startTime = time.Now() }

func metricsHandler(w http.ResponseWriter, _ *http.Request) {
	uptime := int(time.Since(startTime).Seconds())
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "# HELP bounce_uptime_seconds Uptime in seconds\n# TYPE bounce_uptime_seconds gauge\nbounce_uptime_seconds %d\n", uptime)
	var m runtime.MemStats
runtime.ReadMemStats(&m)
	fmt.Fprintf(w, "# HELP bounce_goroutines Number of goroutines\n# TYPE bounce_goroutines gauge\nbounce_goroutines %d\n", runtime.NumGoroutine())
	fmt.Fprintf(w, "# HELP bounce_memory_bytes Allocated memory\n# TYPE bounce_memory_bytes gauge\nbounce_memory_bytes %d\n", m.Alloc)
}
