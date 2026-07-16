package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/mefrraz/bounce/docs"
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

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	dbPath := filepath.Join(dataDir, "bounce.db")
	store, err := cache.NewStore(dbPath)
	if err != nil { log.Fatalf("cache: %v", err) }
	defer store.Close()

	client := httpclient.New()
	defer client.Stop()

	fpb := fpbapi.New(client, store)
	hub := ws.NewHub(nil, nil)

	sched := scheduler.New(
		func(internalID string) (*models.Game, error) {
			detail, err := fpb.GetGame(internalID)
			if err != nil { return nil, err }
			return &detail.Game, nil
		},
		func() ([]models.Game, error) { return nil, nil },
		func(game models.Game) {
			et := "score_update"
			if game.Status == "FINALIZADO" { et = "game_finished" }
			hub.Broadcast(game.ID, ws.Event{Type: et, Data: game})
		},
	)

	hub.SetCallbacks(
		func(gameID string) { sched.ScheduleGameNow(gameID) },
		func(gameID string) { sched.UnscheduleGame(gameID) },
	)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"}, AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
		AllowCredentials: false, MaxAge: 86400,
	}))

	r.Get("/health", apihandler.Health)
	r.Get("/test", apihandler.TestPage)
	r.Get("/app", apihandler.AppPage)
	r.Get("/metrics", metricsHandler)
	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/index.html", http.StatusMovedPermanently)
	})
	r.Get("/docs/*", httpSwagger.WrapHandler)

	handler := apihandler.NewHandler(fpb)
	handler.RegisterRoutes(r)
	hub.RegisterRoutes(r)

	insights := apihandler.NewInsightsHandler()
	insights.RegisterRoutes(r)

	sched.Start()
	defer sched.Stop()

	addr := ":" + port
	srv := &http.Server{Addr: addr, Handler: r}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("starting", "version", apihandler.Version, "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
}

func metricsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("# HELP bounce_uptime_seconds Uptime in seconds\n# TYPE bounce_uptime_seconds counter\nbounce_uptime_seconds 0\n"))
}
