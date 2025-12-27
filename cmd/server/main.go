package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"project/internal/api"
	"project/internal/db"
)

func main() {
	cfg := api.Config{
		Addr:              getEnv("HTTP_ADDR", ":8080"),
		MetadataDBPath:    getEnv("DB_METADATA_PATH", "/data/spotify_metadata.sqlite3"),
		PlaylistsDBPath:   getEnv("DB_PLAYLISTS_PATH", "/data/spotify_playlists.sqlite3"),
		TrackURLsDBPath:   getEnv("DB_TRACK_URLS_PATH", "/data/spotify_track_urls.sqlite3"),
		SpotifyBaseURL:    getEnv("SPOTIFY_BASE_URL", "https://api.spotify.com"),
		TrackfileUpstream: os.Getenv("TRACKFILE_UPSTREAM_URL"),
		AllowOrigin:       os.Getenv("CORS_ALLOW_ORIGIN"),
		ProxyTimeout:      getDurationEnv("PROXY_TIMEOUT", 30*time.Second),
		ReadHeaderTimeout: getDurationEnv("READ_HEADER_TIMEOUT", 10*time.Second),
		IdleTimeout:       getDurationEnv("IDLE_TIMEOUT", 60*time.Second),
	}

	ctx := context.Background()
	store, err := db.Open(ctx, cfg.MetadataDBPath, cfg.PlaylistsDBPath, cfg.TrackURLsDBPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer store.Close()

	h := api.NewHandler(cfg, store)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           h,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	log.Printf("listening on %s", cfg.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
