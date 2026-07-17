package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"golang.org/x/crypto/acme/autocert"

	apihandler "github.com/mefrraz/bounce/internal/api"
	"github.com/mefrraz/bounce/internal/cache"
	"github.com/mefrraz/bounce/internal/docs"
	"github.com/mefrraz/bounce/internal/fpbapi"
	"github.com/mefrraz/bounce/internal/httpclient"
	"github.com/mefrraz/bounce/internal/metrics"
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

	rateLimit := 100
	if rlEnv := os.Getenv("BOUNCE_RATE_LIMIT"); rlEnv != "" {
		if n, err := strconv.Atoi(rlEnv); err == nil && n > 0 { rateLimit = n }
	}
	tlsDomain := os.Getenv("BOUNCE_TLS_DOMAIN")
	tlsCache := os.Getenv("BOUNCE_TLS_CACHE")

	tuiMode := os.Getenv("BOUNCE_TUI") == "true"

	store, err := cache.NewStore(filepath.Join(dataDir, "bounce.db"))
bouncedb = store
	if err != nil { log.Fatalf("cache: %v", err) }
	defer store.Close()

	metrics.SetStore(store)
	metrics.LoadHistory()

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
			fireWebhook(et, g)
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
router = r
r.Use(middleware.Recoverer, middleware.RealIP, middleware.Compress(5))
quiet := os.Getenv("BOUNCE_QUIET") != ""
if !quiet && !tuiMode {
	r.Use(prettyLogger)
}
r.Use(cors.Handler(cors.Options{AllowedOrigins: []string{corsOrigin}, AllowedMethods: []string{"GET", "POST", "OPTIONS"}, AllowedHeaders: []string{"Content-Type", "Authorization"}, AllowCredentials: false, MaxAge: 86400}))

rl := newRateLimiter(rateLimit, time.Minute)
r.Use(rl.middleware)

r.Get("/test", apihandler.TestPage)
r.Get("/", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/dashboard", 302) })
r.Get("/health", healthHandler)
r.Get("/docs", docs.Handler)
r.Get("/metrics", metricsHandler)
r.Get("/api/metrics/history", metrics.HistoryHandler)
r.Get("/api/metrics/history/simple", metrics.HistoryHandlerSimple)
r.Get("/dashboard", metrics.DashboardHandler)
r.Post("/api/batch", batchHandler)

	apihandler.NewHandler(fpb).RegisterRoutes(r)
	hub.RegisterRoutes(r)
ws.RegisterDashboardRoute(r)
	apihandler.NewInsightsHandler().RegisterRoutes(r)

	sched.Start()
	metrics.StartRecording()
	go metricsBroadcaster()

	if tuiMode {
		go func() { fpb.GetCompetitions(); fpb.GetStandings("10902") }()
		runTUI(port, r)
		return
	}

	go func() { fpb.GetCompetitions(); fpb.GetStandings("10902"); slog.Info("pre-warm complete") }()
	srv := &http.Server{Addr: ":" + port, Handler: r}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

go func() {
		if tlsDomain != "" {
			if tlsCache == "" { tlsCache = filepath.Join(dataDir, "autocert") }
			os.MkdirAll(tlsCache, 0700)
			m := &autocert.Manager{
				Cache:      autocert.DirCache(tlsCache),
				Prompt:     autocert.AcceptTOS,
				HostPolicy: autocert.HostWhitelist(tlsDomain),
			}
			srv.TLSConfig = &tls.Config{GetCertificate: m.GetCertificate}
			srv.Addr = ":443"
			go func() { _ = http.ListenAndServe(":80", m.HTTPHandler(nil)) }()
			slog.Info("starting", "version", apihandler.Version, "tls_domain", tlsDomain, "addr", ":443")
			if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed { log.Fatalf("server: %v", err) }
		} else {
			slog.Info("starting", "version", apihandler.Version, "addr", srv.Addr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed { log.Fatalf("server: %v", err) }
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")
	metrics.RecordSnapshot() // save final state
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
}

func prettyLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		fmt.Printf("\033[90m[%s]\033[0m \033[36m%s\033[0m %s → \033[%dm%d\033[0m %v\n",
			time.Now().Format("15:04:05"),
			r.Method,
			r.URL.Path,
			statusColor(ww.Status()),
			ww.Status(),
			time.Since(start).Round(time.Microsecond),
		)
	})
}

func statusColor(code int) int {
	if code < 300 { return 32 } // green
	if code < 400 { return 33 } // yellow
	return 31 // red
}

var startTime = time.Now()

func init() { startTime = time.Now() }

func metricsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"version":         apihandler.Version,
		"uptime_seconds":  int(time.Since(startTime).Seconds()),
		"goroutines":      runtime.NumGoroutine(),
		"memory_alloc_mb": float64(m.Alloc) / 1024 / 1024,
		"requests":        metrics.RequestsTotal,
		"cache_hits":      metrics.CacheHitsTotal,
		"cache_misses":    metrics.CacheMissesTotal,
		"fpb_requests":    metrics.FPBRequestsTotal,
		"rate_limited":    metrics.RateLimitedTotal,
	})
}

func metricsBroadcaster() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		data := map[string]interface{}{
			"requests":       metrics.RequestsTotal,
			"cache_hits":      metrics.CacheHitsTotal,
			"cache_misses":    metrics.CacheMissesTotal,
			"fpb_requests":    metrics.FPBRequestsTotal,
			"rate_limited":    metrics.RateLimitedTotal,
			"goroutines":      runtime.NumGoroutine(),
			"uptime_seconds":  int(time.Since(startTime).Seconds()),
		}
		ws.BroadcastMetrics(data)
	}
}

// ── TUI mode ──

func runTUI(port string, handler http.Handler) {
	fmt.Print("\033[2J\033[?25l")
	defer fmt.Print("\033[?25h")

	srv := &http.Server{Addr: ":" + port, Handler: handler}
	go func() { srv.ListenAndServe() }()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastReq uint64
	for range ticker.C {
		reqs := metrics.RequestsTotal
		rps := reqs - lastReq
		lastReq = reqs
		ch := metrics.CacheHitsTotal
		cm := metrics.CacheMissesTotal
		total := ch + cm
		rate := 0
		if total > 0 { rate = int(ch * 100 / total) }
		uptime := time.Since(startTime).Round(time.Second)

		// Header
		fmt.Printf("\033[H\033[1;38;5;208m  Bounce %s  \033[32m● online\033[0m  \033[90m:%s\033[0m\n", apihandler.Version, port)

		// Left side: metrics
		fmt.Printf("\033[32m  Requests:\033[0m %d  \033[90m│\033[0m  \033[36mCache:\033[0m %d%%  \033[90m│\033[0m  \033[33mFPB Reqs:\033[0m %d  \033[90m│\033[0m  \033[31mLimited:\033[0m %d\n",
			reqs, rate, metrics.FPBRequestsTotal, metrics.RateLimitedTotal)
		fmt.Printf("  \033[35mGoroutines:\033[0m %d  \033[90m│\033[0m  \033[34mReqs/sec:\033[0m %d  \033[90m│\033[0m  \033[37mUptime:\033[0m %v\n",
			runtime.NumGoroutine(), rps*2, uptime)

		// Footer
		fmt.Printf("\n  \033[90mPress Ctrl+C to stop\033[0m\n\033[J")
	}
}

var (
	router   chi.Router
	bouncedb *cache.Store
)

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	dbOk := false
	if bouncedb != nil {
		dbOk = bouncedb.Ping()
	}
	w.Header().Set("Content-Type", "application/json")
	status := "ok"
	if !dbOk { status = "degraded" }
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    status,
		"version":   apihandler.Version,
		"db_ok":     dbOk,
		"uptime":    time.Since(startTime).String(),
	})
}

var webhookURL = os.Getenv("BOUNCE_WEBHOOK_URL")

type batchReq struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

func batchHandler(w http.ResponseWriter, req *http.Request) {
	var batch []batchReq
	if err := json.NewDecoder(req.Body).Decode(&batch); err != nil || len(batch) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid body, expected [{\"method\":\"GET\",\"path\":\"/api/...\"},...]"})
		return
	}
	var results []map[string]interface{}
	for _, br := range batch {
		method := br.Method
		if method == "" { method = "GET" }
		subReq, _ := http.NewRequest(method, "http://localhost"+br.Path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, subReq)
		var body interface{}
		json.Unmarshal(rec.Body.Bytes(), &body)
		results = append(results, map[string]interface{}{
			"path":   br.Path,
			"status": rec.Code,
			"body":   body,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func fireWebhook(event string, data interface{}) {
	if webhookURL == "" { return }
	payload, _ := json.Marshal(map[string]interface{}{"event": event, "data": data, "time": time.Now().UTC()})
	go func() { http.Post(webhookURL, "application/json", bytes.NewReader(payload)) }()
}
