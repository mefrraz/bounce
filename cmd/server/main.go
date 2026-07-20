package main

import (
	"bytes"
	"context"
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"sync"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"golang.org/x/crypto/acme/autocert"
_ "github.com/andybalholm/brotli"

	apihandler "github.com/mefrraz/bounce/internal/api"
	"github.com/mefrraz/bounce/internal/cache"
	"github.com/mefrraz/bounce/internal/clubs"
	"github.com/mefrraz/bounce/internal/docs"
	"github.com/mefrraz/bounce/internal/fpbapi"
	"github.com/mefrraz/bounce/internal/httpclient"
	"github.com/mefrraz/bounce/internal/metrics"
	"github.com/mefrraz/bounce/internal/models"
	"github.com/mefrraz/bounce/internal/scheduler"
	"github.com/mefrraz/bounce/internal/ws"
	"github.com/mefrraz/bounce/internal/elo"
)

var adminLoginHTML = []byte(`<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"><title>Bounce Admin · Login</title><style>:root{--bg:#09090b;--card:#18181b;--border:rgba(255,255,255,0.1);--text:#f4f4f5;--muted:#71717a;--accent:#ff6b35}body{background:var(--bg);color:var(--text);font-family:system-ui,sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0}.card{background:var(--card);border:1px solid var(--border);border-radius:16px;padding:32px;width:360px;text-align:center}h1{font-size:22px;margin:0 0 8px}.sub{color:var(--muted);font-size:14px;margin:0 0 20px}input{width:100%;padding:10px 14px;border:1px solid var(--border);border-radius:10px;background:var(--bg);color:var(--text);font-size:14px;box-sizing:border-box;outline:none}input:focus{border-color:var(--accent)}button{margin-top:12px;width:100%;padding:10px;background:var(--accent);color:#fff;border:none;border-radius:10px;font-size:14px;font-weight:600;cursor:pointer}.err{color:#ef4444;font-size:13px;margin-top:8px;display:none}</style></head><body><div class="card"><h1>🔐 Bounce Admin</h1><p class="sub">Enter admin token to continue</p><input type="password" id="token" placeholder="BOUNCE_ADMIN_TOKEN"><button onclick="login()">Login</button><p class="err" id="err">Invalid token</p></div><script>async function login(){const r=await fetch('/admin/login',{method:'POST',body:JSON.stringify({token:document.getElementById('token').value})});if(r.ok)location.reload();else document.getElementById('err').style.display='block'}</script></body></html>`)

var adminPageHTML = []byte(`<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"><title>Bounce Admin · Clubs</title><style>:root{--bg:#09090b;--card:#18181b;--border:rgba(255,255,255,0.1);--text:#f4f4f5;--muted:#71717a;--accent:#ff6b35;--green:#22c55e}body{background:var(--bg);color:var(--text);font-family:system-ui,sans-serif;margin:0;padding:16px 24px}.top{display:flex;align-items:center;justify-content:space-between;margin-bottom:16px;position:sticky;top:0;background:var(--bg);padding:12px 0;z-index:10}h1{font-size:22px;margin:0}.top-l{display:flex;align-items:center;gap:12px}.btn{padding:8px 16px;border-radius:10px;font-size:13px;font-weight:600;cursor:pointer;border:none;transition:all .15s}.btn-accent{background:var(--accent);color:#fff}.btn-accent:hover{opacity:.85}.btn-outline{background:transparent;border:1px solid var(--border);color:var(--text)}.btn-outline:hover{background:rgba(255,255,255,0.05)}input.search{width:260px;padding:8px 12px;border:1px solid var(--border);border-radius:10px;background:var(--card);color:var(--text);font-size:13px;outline:none}input.search:focus{border-color:var(--accent)}table{width:100%;border-collapse:collapse;font-size:13px}th{text-align:left;color:var(--muted);font-weight:600;font-size:11px;text-transform:uppercase;letter-spacing:.04em;padding:8px 10px;border-bottom:1px solid var(--border)}td{padding:8px 10px;border-bottom:1px solid rgba(255,255,255,0.04)}.color-dot{display:inline-block;width:14px;height:14px;border-radius:4px;vertical-align:middle;margin-right:6px;border:1px solid rgba(255,255,255,0.15)}.logo-img{width:28px;height:28px;object-fit:contain;border-radius:4px;background:rgba(255,255,255,0.05)}.editable{border:none;background:transparent;color:var(--text);font-size:13px;padding:2px 4px;border-radius:4px;width:100%;box-sizing:border-box}.editable:focus{background:rgba(255,255,255,0.05);outline:1px solid var(--accent)}.status{font-size:12px;color:var(--muted)}.status.ok{color:var(--green)}</style></head><body><div class="top"><div class="top-l"><h1>🏀 Bounce Admin</h1><span class="status" id="status"></span></div><div class="top-l"><input class="search" id="search" placeholder="Search clubs..." oninput="render()"><button class="btn btn-accent" onclick="refresh()">🔄 Refresh from FPB</button><a href="/dashboard" class="btn btn-outline" style="text-decoration:none">📊 Dashboard</a></div></div><table><thead><tr><th>ID</th><th>Logo</th><th>Name</th><th>Short Name</th><th>Color</th><th>Priority</th></tr></thead><tbody id="tbody"></tbody></table><script>let clubs=[];async function load(){const r=await fetch('/api/clubs');clubs=await r.json();clubs.sort((a,b)=>(b.priority||0)-(a.priority||0)||a.name.localeCompare(b.name));document.getElementById('status').textContent=clubs.length+' clubs';render()}function render(){const q=(document.getElementById('search').value||'').toLowerCase();const filter=q?clubs.filter(c=>c.name.toLowerCase().includes(q)||(c.short_name||'').toLowerCase().includes(q)||String(c.id).includes(q)):clubs.slice(0,50);document.getElementById('tbody').innerHTML=filter.map(c=>'<tr><td>'+c.id+'</td><td>'+(c.logo_url?'<img class=logo-img src='+c.logo_url+'>':'—')+'</td><td><input class=editable value="'+esc(c.name)+'" onchange=save('+c.id+',"name",this.value)></td><td><input class=editable value="'+esc(c.short_name||'')+'" onchange=save('+c.id+',"short_name",this.value)></td><td><span class=color-dot style=background:'+esc(c.primary_color||'#7C3AED')+'></span><input class=editable value="'+esc(c.primary_color||'')+'" style=width:80px onchange=save('+c.id+',"primary_color",this.value)></td><td><input class=editable value="'+(c.priority||0)+'" style=width:50px type=number onchange=save('+c.id+',"priority",this.value)></td></tr>').join('')}function esc(s){return String(s||'').replace(/&/g,'&amp;').replace(/"/g,'&quot;').replace(/</g,'&lt;').replace(/>/g,'&gt;')}function save(id,field,val){fetch('/api/clubs/'+id,{method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify({[field]:field==='priority'?parseInt(val)||0:val})}).then(()=>document.getElementById('status').textContent='Saved '+id+' '+field).catch(()=>document.getElementById('status').textContent='Error saving')}async function refresh(){document.getElementById('status').textContent='Refreshing...';const r=await fetch('/api/clubs/refresh',{method:'POST'});const d=await r.json();document.getElementById('status').textContent=d.updated+' updated, '+d.added+' added, '+d.errors+' errors';if(d.added>0)load()}load()</script></body></html>`)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	port := os.Getenv("BOUNCE_PORT")
	if port == "" { port = "3001" }
	dataDir := os.Getenv("BOUNCE_DATA_DIR")
	if dataDir == "" { dataDir = "/data" }
	if err := os.MkdirAll(dataDir, 0755); err != nil { log.Fatalf("data dir: %v", err) }

	// Initialize clubs data (load from disk or seed from embedded)
	if err := clubs.Init(dataDir); err != nil {
		slog.Warn("clubs init: "+err.Error()+" — seeding from embedded data")
		clubs.RefreshFromFPB()
	}
	clubs.StartDailyRefresh()
	go clubs.EnsureColors() // extract logo colors in background

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

	// Import/Srape/ELO based on BOUNCE_MODE
	bounceMode := os.Getenv("BOUNCE_MODE")
	if bounceMode == "" { bounceMode = "import" }

	if bounceMode != "off" {
		go func() {
			if bounceMode == "import" {
				store.ImportGamesFromSupabase()
			} else if bounceMode == "scrape" {
				log.Printf("[scrape] mode=scrape — scraping all clubs for current season")
				fpb.ScrapeAllClubs(cache.CurrentSeason())
			}

			// Calculate ELO for all seasons with games (regardless of mode)
			rows, err := store.DB().Query("SELECT DISTINCT season FROM games ORDER BY season")
			if err != nil { log.Printf("[elo] query seasons: %v", err); return }
			defer rows.Close()
			var allSeasons []string
			for rows.Next() { var s string; if rows.Scan(&s) == nil && s != "" { allSeasons = append(allSeasons, s) } }

			eloStore := elo.NewStore(store.DB())
			for _, s := range allSeasons {
				if eloStore.HasSeason(s) { continue }
				log.Printf("[elo] calculating %s", s)
				if err := fpb.RecalculateELO(s); err != nil { log.Printf("[elo] %s: %v", s, err) }
			}
		}()
	}

	hub := ws.NewHub(nil, nil)

	sched := scheduler.New(
		func(id string) (*models.Game, error) { d, e := fpb.GetGame(id); if e != nil { return nil, e }; return &d.Game, nil },
		func() ([]models.Game, error) { comps, _ := fpb.GetCompetitions("", ""); var t []models.Game; for _, c := range comps { g, _ := fpb.GetGamesByCompetition(c.ID, cache.CurrentSeason()); for _, gm := range g { if cache.IsToday(gm.Date) { t = append(t, gm) } } }; slog.Info("daily refresh", "games_today", len(t)); return t, nil },
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
	if tuiMode {
		r.Use(tuiLogger)
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
r.Post("/api/metrics/reset", metricsResetHandler)
r.Get("/dashboard", metrics.DashboardHandler)
r.Post("/api/batch", batchHandler)
r.Get("/api/clubs", clubsHandler)
r.Post("/api/clubs/refresh", clubsRefreshHandler)
r.Patch("/api/clubs/{id}", clubsPatchHandler)
r.Get("/admin", adminPageHandler)
r.Post("/admin/login", adminLoginHandler)

	apiHandler := apihandler.NewHandler(fpb)
	apiHandler.RegisterRoutes(r)
	hub.RegisterRoutes(r)
ws.RegisterDashboardRoute(r)
	apihandler.NewInsightsHandler().RegisterRoutes(r)

	sched.Start()
	metrics.StartRecording()
	go metricsBroadcaster()

	// Daily scraper + ELO recalculation (aligns to 3am)
	apiHandler.StartDailyScrapeAndELO()

	if tuiMode {
		go func() { fpb.GetCompetitions("", ""); fpb.GetStandings("10902") }()
		runTUI(port, r)
		return
	}

	go func() { fpb.GetCompetitions("", ""); fpb.GetStandings("10902"); slog.Info("pre-warm complete") }()
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
		"reqs_per_sec":    metrics.ReqRate(),
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

	go listenKeys()
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
		fmt.Printf("\033[2J\033[H\033[1;38;5;208m  Bounce %s  \033[32m● online\033[0m  \033[90m:%s\033[0m\n", apihandler.Version, port)

		// Left side: metrics
		fmt.Printf("\033[32m  Requests:\033[0m %d  \033[90m│\033[0m  \033[36mCache:\033[0m %d%%  \033[90m│\033[0m  \033[33mFPB Reqs:\033[0m %d  \033[90m│\033[0m  \033[31mLimited:\033[0m %d\n",
			reqs, rate, metrics.FPBRequestsTotal, metrics.RateLimitedTotal)
		fmt.Printf("  \033[35mGoroutines:\033[0m %d  \033[90m│\033[0m  \033[34mReqs/sec:\033[0m %d  \033[90m│\033[0m  \033[37mUptime:\033[0m %v\n",
			runtime.NumGoroutine(), rps*2, uptime)

		// Recent requests
		for i := 0; i < 8; i++ {
			idx := (tuiReqIdx - 1 - i + 8) % 8
			if tuiReqLog[idx] != "" {
				fmt.Printf("  %s\n", tuiReqLog[idx])
			}
		}

		// Footer
		fmt.Printf("\n  \033[90mPress Ctrl+C to stop  R=reset\033[0m\n\033[J")
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

func metricsResetHandler(w http.ResponseWriter, _ *http.Request) {
	metrics.ResetAll()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","message":"metrics reset"}`))
}

func clubsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clubs.All())
}

var adminToken = os.Getenv("BOUNCE_ADMIN_TOKEN")

func checkAdmin(r *http.Request) bool {
	if adminToken == "" { return false }
	if cookie, err := r.Cookie("bounce_admin"); err == nil && cookie.Value == adminToken {
		return true
	}
	auth := r.Header.Get("Authorization")
	return auth == "Bearer "+adminToken || auth == adminToken
}

func clubsRefreshHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdmin(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}
	updated, added, errs := clubs.RefreshFromFPB()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"updated": updated, "added": added, "errors": errs, "total": len(clubs.All()),
	})
}

func adminLoginHandler(w http.ResponseWriter, r *http.Request) {
	if adminToken == "" {
		http.Error(w, "Admin not configured", http.StatusForbidden)
		return
	}
	var body struct{ Token string }
	json.NewDecoder(r.Body).Decode(&body)
	if body.Token != adminToken {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid token"})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: "bounce_admin", Value: adminToken, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 86400,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func adminPageHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdmin(r) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(adminLoginHTML)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(adminPageHTML)
}

func clubsPatchHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdmin(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.Atoi(idStr)
	var body map[string]interface{}
	json.NewDecoder(r.Body).Decode(&body)
	name, _ := body["name"].(string)
	shortName, _ := body["short_name"].(string)
	primaryColor, _ := body["primary_color"].(string)
	logoURL, _ := body["logo_url"].(string)
	priority := 0
	if p, ok := body["priority"].(float64); ok { priority = int(p) }
	if err := clubs.UpdateClub(id, name, shortName, primaryColor, logoURL, priority); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ── TUI keyboard handler ──
func listenKeys() {
	var buf [1]byte
	for {
		os.Stdin.Read(buf[:])
		if buf[0] == 'r' || buf[0] == 'R' {
			metrics.ResetAll()
		}
	}
}

// ── TUI request log ──
var tuiReqLog [8]string
var tuiReqIdx int
var tuiReqMu sync.Mutex

func tuiLogReq(method, path string, code int, dur time.Duration) {
	tuiReqMu.Lock()
	c := "32"; if code >= 400 { c = "31" } else if code >= 300 { c = "33" }
	tuiReqLog[tuiReqIdx%8] = fmt.Sprintf("\033[%sm%3d\033[0m \033[36m%s\033[0m \033[90m%s\033[0m %v", c, code, method, path, dur.Round(time.Microsecond))
	tuiReqIdx++
	tuiReqMu.Unlock()
}

func tuiLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		tuiLogReq(r.Method, r.URL.Path, ww.Status(), time.Since(start))
	})
}
