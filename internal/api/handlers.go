package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"project/internal/db"
	"project/internal/httpx"
)

type API struct {
	cfg   Config
	store *db.Store
}

func (api *API) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/v1" && r.URL.Path != "/v1/" {
		http.NotFound(w, r)
		return
	}
	payload := map[string]any{
		"name":    "spotify-query-api",
		"version": "v1",
		"routes": []string{
			"GET /v1/health",
			"GET /v1/tracks/search",
			"GET /v1/tracks/{id}",
			"GET /v1/albums/search",
			"GET /v1/albums/{id}",
			"GET /v1/artists/search",
			"GET /v1/artists/{id}",
			"GET /v1/playlists/{id}",
			"ANY /v1/spotify/*",
			"ANY /v1/trackfiles/*",
		},
	}
	httpx.WriteJSON(w, http.StatusOK, payload)
}

func (api *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := api.store.DB.PingContext(ctx); err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "db_unavailable", err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (api *API) handleTrackSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("query"))
	artist := strings.TrimSpace(r.URL.Query().Get("artist"))
	album := strings.TrimSpace(r.URL.Query().Get("album"))
	genre := strings.TrimSpace(r.URL.Query().Get("genre"))

	limit, offset := parseLimitOffset(r.URL.Query())
	sort := strings.TrimSpace(r.URL.Query().Get("sort"))

	rows, err := queryTrackSearch(r.Context(), api.store.DB, TrackSearchParams{
		Query:  q,
		Artist: artist,
		Album:  album,
		Genre:  genre,
		Limit:  limit,
		Offset: offset,
		Sort:   sort,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	payload := map[string]any{
		"paging": map[string]any{
			"limit":  limit,
			"offset": offset,
		},
		"data": rows,
	}
	httpx.WriteJSON(w, http.StatusOK, payload)
}

func (api *API) handleTrackByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/tracks/")
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "/") {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "track id is required")
		return
	}

	track, err := queryTrackDetail(r.Context(), api.store.DB, id)
	if err != nil {
		if err == sql.ErrNoRows {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "track not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, track)
}

func (api *API) handleAlbumSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("query"))
	artist := strings.TrimSpace(r.URL.Query().Get("artist"))
	limit, offset := parseLimitOffset(r.URL.Query())

	rows, err := queryAlbumSearch(r.Context(), api.store.DB, AlbumSearchParams{
		Query:  q,
		Artist: artist,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	payload := map[string]any{
		"paging": map[string]any{
			"limit":  limit,
			"offset": offset,
		},
		"data": rows,
	}
	httpx.WriteJSON(w, http.StatusOK, payload)
}

func (api *API) handleAlbumByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/albums/")
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "/") {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "album id is required")
		return
	}

	album, err := queryAlbumDetail(r.Context(), api.store.DB, id)
	if err != nil {
		if err == sql.ErrNoRows {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "album not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, album)
}

func (api *API) handleArtistSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("query"))
	genre := strings.TrimSpace(r.URL.Query().Get("genre"))
	limit, offset := parseLimitOffset(r.URL.Query())

	rows, err := queryArtistSearch(r.Context(), api.store.DB, ArtistSearchParams{
		Query:  q,
		Genre:  genre,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	payload := map[string]any{
		"paging": map[string]any{
			"limit":  limit,
			"offset": offset,
		},
		"data": rows,
	}
	httpx.WriteJSON(w, http.StatusOK, payload)
}

func (api *API) handleArtistByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/artists/")
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "/") {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "artist id is required")
		return
	}

	artist, err := queryArtistDetail(r.Context(), api.store.DB, id)
	if err != nil {
		if err == sql.ErrNoRows {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "artist not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, artist)
}

func (api *API) handlePlaylistByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/playlists/")
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "/") {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_id", "playlist id is required")
		return
	}

	limit, offset := parseLimitOffset(r.URL.Query())
	playlist, err := queryPlaylistDetailWithParams(r.Context(), api.store.DB, id, PlaylistParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "playlist not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "query_failed", err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, playlist)
}

func (api *API) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if api.cfg.AllowOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", api.cfg.AllowOrigin)
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func parseLimitOffset(values url.Values) (int, int) {
	limit := parseInt(values.Get("limit"), 25)
	offset := parseInt(values.Get("offset"), 0)

	if limit <= 0 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	return limit, offset
}

func parseInt(val string, fallback int) int {
	if val == "" {
		return fallback
	}
	if parsed, err := strconv.Atoi(val); err == nil {
		return parsed
	}
	return fallback
}

func unixToTime(value int64) *time.Time {
	if value == 0 {
		return nil
	}
	if value > 1e12 {
		t := time.UnixMilli(value).UTC()
		return &t
	}
	t := time.Unix(value, 0).UTC()
	return &t
}

func parseBoolInt(val int64) bool {
	return val != 0
}

func parseJSONList(val string) []string {
	if strings.TrimSpace(val) == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(val), &out); err == nil {
		return out
	}
	parts := strings.Split(val, ",")
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func formatQueryLike(input string) string {
	if input == "" {
		return ""
	}
	return fmt.Sprintf("%%%s%", strings.ReplaceAll(input, "%", ""))
}
