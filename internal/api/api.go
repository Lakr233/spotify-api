package api

import (
	"net/http"
	"time"

	"project/internal/db"
)

type Config struct {
	Addr              string
	MetadataDBPath    string
	PlaylistsDBPath   string
	TrackURLsDBPath   string
	SpotifyBaseURL    string
	TrackfileUpstream string
	AllowOrigin       string
	ProxyTimeout      time.Duration
	ReadHeaderTimeout time.Duration
	IdleTimeout       time.Duration
}

func NewHandler(cfg Config, store *db.Store) http.Handler {
	api := &API{
		cfg:   cfg,
		store: store,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", api.handleRoot)
	mux.HandleFunc("/v1", api.handleRoot)
	mux.HandleFunc("/v1/", api.handleRoot)
	mux.HandleFunc("/v1/health", api.handleHealth)

	mux.HandleFunc("/v1/tracks/search", api.handleTrackSearch)
	mux.HandleFunc("/v1/tracks/", api.handleTrackByID)

	mux.HandleFunc("/v1/albums/search", api.handleAlbumSearch)
	mux.HandleFunc("/v1/albums/", api.handleAlbumByID)

	mux.HandleFunc("/v1/artists/search", api.handleArtistSearch)
	mux.HandleFunc("/v1/artists/", api.handleArtistByID)

	mux.HandleFunc("/v1/playlists/", api.handlePlaylistByID)

	mux.Handle("/v1/spotify/", api.spotifyProxy())
	mux.Handle("/v1/trackfiles/", api.trackfileProxy())

	return api.withCORS(mux)
}
