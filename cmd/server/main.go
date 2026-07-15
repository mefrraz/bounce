package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	apihandler "github.com/mefrraz/bounce/internal/api"
	"github.com/mefrraz/bounce/internal/cache"
	"github.com/mefrraz/bounce/internal/fpbapi"
	"github.com/mefrraz/bounce/internal/httpclient"
)

func main() {
	port := os.Getenv("BOUNCE_PORT")
	if port == "" {
		port = "3001"
	}
	dataDir := os.Getenv("BOUNCE_DATA_DIR")
	if dataDir == "" {
		dataDir = "/data"
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	dbPath := filepath.Join(dataDir, "bounce.db")
	store, err := cache.NewStore(dbPath)
	if err != nil {
		log.Fatalf("cache: %v", err)
	}
	defer store.Close()

	client := httpclient.New()
	defer client.Stop()

	fpb := fpbapi.New(client, store)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           86400,
	}))

	r.Get("/health", apihandler.Health)
	r.Get("/test", apihandler.TestPage)

	handler := apihandler.NewHandler(fpb)
	handler.RegisterRoutes(r)

	addr := ":" + port
	log.Printf("Bounce v0.3.0 starting on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
