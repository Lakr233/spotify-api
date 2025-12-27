package api

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
)

type TrackSearchParams struct {
	Query  string
	Artist string
	Album  string
	Genre  string
	Limit  int
	Offset int
	Sort   string
}

type AlbumSearchParams struct {
	Query  string
	Artist string
	Limit  int
	Offset int
}

type ArtistSearchParams struct {
	Query  string
	Genre  string
	Limit  int
	Offset int
}

type PlaylistParams struct {
	Limit  int
	Offset int
}

func queryTrackSearch(ctx context.Context, db *sql.DB, params TrackSearchParams) ([]TrackSummary, error) {
	where := []string{"1=1"}
	args := []any{}

	if params.Query != "" {
		where = append(where, "t.name LIKE ?")
		args = append(args, formatQueryLike(params.Query))
	}
	if params.Artist != "" {
		where = append(where, "EXISTS (SELECT 1 FROM track_artists ta JOIN artists ar ON ar.rowid = ta.artist_rowid WHERE ta.track_rowid = t.rowid AND ar.name LIKE ?)")
		args = append(args, formatQueryLike(params.Artist))
	}
	if params.Album != "" {
		where = append(where, "a.name LIKE ?")
		args = append(args, formatQueryLike(params.Album))
	}
	if params.Genre != "" {
		where = append(where, "EXISTS (SELECT 1 FROM track_artists ta JOIN artist_genres ag ON ag.artist_rowid = ta.artist_rowid WHERE ta.track_rowid = t.rowid AND ag.genre LIKE ?)")
		args = append(args, formatQueryLike(params.Genre))
	}

	sortClause := "t.popularity DESC"
	switch strings.ToLower(params.Sort) {
	case "name":
		sortClause = "t.name COLLATE NOCASE ASC"
	case "release_date":
		sortClause = "a.release_date DESC"
	case "recent":
		sortClause = "t.fetched_at DESC"
	case "popularity", "":
		// default
	default:
		sortClause = "t.popularity DESC"
	}

	query := fmt.Sprintf(`
SELECT
	 t.rowid,
	 t.id,
	 t.name,
	 t.popularity,
	 t.duration_ms,
	 t.explicit,
	 t.preview_url,
	 t.external_id_isrc,
	 a.id,
	 a.name,
	 a.album_type,
	 a.release_date,
	 a.total_tracks,
	 a.popularity
FROM tracks t
JOIN albums a ON a.rowid = t.album_rowid
WHERE %s
ORDER BY %s
LIMIT ? OFFSET ?`, strings.Join(where, " AND "), sortClause)

	args = append(args, params.Limit, params.Offset)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		trackRowIDs []int64
		results     []TrackSummary
	)

	for rows.Next() {
		var (
			rowID        int64
			trackID      string
			name         string
			popularity   int
			durationMS   int
			explicit     int64
			previewURL   sql.NullString
			externalISRC sql.NullString
			albumID      string
			albumName    string
			albumType    string
			releaseDate  string
			totalTracks  int
			albumPop     int
		)

		if err := rows.Scan(&rowID, &trackID, &name, &popularity, &durationMS, &explicit, &previewURL, &externalISRC,
			&albumID, &albumName, &albumType, &releaseDate, &totalTracks, &albumPop); err != nil {
			return nil, err
		}

		trackRowIDs = append(trackRowIDs, rowID)
		results = append(results, TrackSummary{
			ID:           trackID,
			Name:         name,
			Popularity:   popularity,
			DurationMS:   durationMS,
			Explicit:     parseBoolInt(explicit),
			PreviewURL:   nullStringPtr(previewURL),
			ExternalISRC: nullStringPtr(externalISRC),
			Album: AlbumSummary{
				ID:          albumID,
				Name:        albumName,
				AlbumType:   albumType,
				ReleaseDate: releaseDate,
				TotalTracks: totalTracks,
				Popularity:  albumPop,
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return results, nil
	}

	artistsByTrack, err := fetchTrackArtists(ctx, db, trackRowIDs)
	if err != nil {
		return nil, err
	}

	for i := range results {
		rowID := trackRowIDs[i]
		results[i].Artists = artistsByTrack[rowID]
	}

	return results, nil
}

func queryTrackDetail(ctx context.Context, db *sql.DB, trackID string) (*TrackDetail, error) {
	var (
		trackRowID       int64
		albumRowID       int64
		name             string
		popularity       int
		durationMS       int
		explicit         int64
		previewURL       sql.NullString
		externalISRC     sql.NullString
		fetchedAt        int64
		albumID          string
		albumName        string
		albumType        string
		releaseDate      string
		albumTotalTracks int
		albumPop         int
		albumLabel       string
		albumReleasePrec string
		albumUPC         sql.NullString
		albumCopyrightC  sql.NullString
		albumCopyrightP  sql.NullString
	)

	query := `
SELECT
	 t.rowid,
	 t.id,
	 t.name,
	 t.popularity,
	 t.duration_ms,
	 t.explicit,
	 t.preview_url,
	 t.external_id_isrc,
	 t.fetched_at,
	 a.rowid,
	 a.id,
	 a.name,
	 a.album_type,
	 a.release_date,
	 a.total_tracks,
	 a.popularity,
	 a.label,
	 a.release_date_precision,
	 a.external_id_upc,
	 a.copyright_c,
	 a.copyright_p
FROM tracks t
JOIN albums a ON a.rowid = t.album_rowid
WHERE t.id = ?`

	if err := db.QueryRowContext(ctx, query, trackID).Scan(
		&trackRowID,
		&trackID,
		&name,
		&popularity,
		&durationMS,
		&explicit,
		&previewURL,
		&externalISRC,
		&fetchedAt,
		&albumRowID,
		&albumID,
		&albumName,
		&albumType,
		&releaseDate,
		&albumTotalTracks,
		&albumPop,
		&albumLabel,
		&albumReleasePrec,
		&albumUPC,
		&albumCopyrightC,
		&albumCopyrightP,
	); err != nil {
		return nil, err
	}

	albumImages, err := fetchAlbumImages(ctx, db, albumRowID)
	if err != nil {
		return nil, err
	}

	artists, err := fetchArtistsForTrack(ctx, db, trackRowID)
	if err != nil {
		return nil, err
	}

	trackFile, err := fetchTrackFile(ctx, db, trackID)
	if err != nil {
		return nil, err
	}

	return &TrackDetail{
		Track: TrackSummary{
			ID:           trackID,
			Name:         name,
			Popularity:   popularity,
			DurationMS:   durationMS,
			Explicit:     parseBoolInt(explicit),
			PreviewURL:   nullStringPtr(previewURL),
			ExternalISRC: nullStringPtr(externalISRC),
			Album: AlbumSummary{
				ID:          albumID,
				Name:        albumName,
				AlbumType:   albumType,
				ReleaseDate: releaseDate,
				TotalTracks: albumTotalTracks,
				Popularity:  albumPop,
				Images:      albumImages,
			},
		},
		Album: AlbumDetail{
			AlbumSummary: AlbumSummary{
				ID:          albumID,
				Name:        albumName,
				AlbumType:   albumType,
				ReleaseDate: releaseDate,
				TotalTracks: albumTotalTracks,
				Popularity:  albumPop,
				Images:      albumImages,
			},
			Label:                albumLabel,
			ReleaseDatePrecision: albumReleasePrec,
			ExternalUPC:          nullStringPtr(albumUPC),
			CopyrightC:           nullStringPtr(albumCopyrightC),
			CopyrightP:           nullStringPtr(albumCopyrightP),
		},
		Artists:   artists,
		TrackFile: trackFile,
		FetchedAt: unixStringPtr(fetchedAt),
	}, nil
}

func queryAlbumSearch(ctx context.Context, db *sql.DB, params AlbumSearchParams) ([]AlbumSummary, error) {
	where := []string{"1=1"}
	args := []any{}
	if params.Query != "" {
		where = append(where, "a.name LIKE ?")
		args = append(args, formatQueryLike(params.Query))
	}
	if params.Artist != "" {
		where = append(where, "EXISTS (SELECT 1 FROM artist_albums aa JOIN artists ar ON ar.rowid = aa.artist_rowid WHERE aa.album_rowid = a.rowid AND ar.name LIKE ?)")
		args = append(args, formatQueryLike(params.Artist))
	}

	query := fmt.Sprintf(`
SELECT
	 a.rowid,
	 a.id,
	 a.name,
	 a.album_type,
	 a.release_date,
	 a.total_tracks,
	 a.popularity
FROM albums a
WHERE %s
ORDER BY a.popularity DESC
LIMIT ? OFFSET ?`, strings.Join(where, " AND "))

	args = append(args, params.Limit, params.Offset)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		albumRowIDs []int64
		results     []AlbumSummary
	)
	for rows.Next() {
		var (
			rowID       int64
			albumID     string
			name        string
			albumType   string
			releaseDate string
			totalTracks int
			popularity  int
		)
		if err := rows.Scan(&rowID, &albumID, &name, &albumType, &releaseDate, &totalTracks, &popularity); err != nil {
			return nil, err
		}
		albumRowIDs = append(albumRowIDs, rowID)
		results = append(results, AlbumSummary{
			ID:          albumID,
			Name:        name,
			AlbumType:   albumType,
			ReleaseDate: releaseDate,
			TotalTracks: totalTracks,
			Popularity:  popularity,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return results, nil
	}

	imagesByAlbum, err := fetchAlbumImagesBatch(ctx, db, albumRowIDs)
	if err != nil {
		return nil, err
	}

	for i := range results {
		results[i].Images = imagesByAlbum[albumRowIDs[i]]
	}

	return results, nil
}

func queryAlbumDetail(ctx context.Context, db *sql.DB, albumID string) (*AlbumDetail, error) {
	var (
		albumRowID       int64
		name             string
		albumType        string
		releaseDate      string
		albumTotalTracks int
		albumPop         int
		label            string
		releasePrecision string
		externalUPC      sql.NullString
		copyrightC       sql.NullString
		copyrightP       sql.NullString
	)

	query := `
SELECT
	 a.rowid,
	 a.id,
	 a.name,
	 a.album_type,
	 a.release_date,
	 a.total_tracks,
	 a.popularity,
	 a.label,
	 a.release_date_precision,
	 a.external_id_upc,
	 a.copyright_c,
	 a.copyright_p
FROM albums a
WHERE a.id = ?`

	if err := db.QueryRowContext(ctx, query, albumID).Scan(
		&albumRowID,
		&albumID,
		&name,
		&albumType,
		&releaseDate,
		&albumTotalTracks,
		&albumPop,
		&label,
		&releasePrecision,
		&externalUPC,
		&copyrightC,
		&copyrightP,
	); err != nil {
		return nil, err
	}

	images, err := fetchAlbumImages(ctx, db, albumRowID)
	if err != nil {
		return nil, err
	}

	artists, err := fetchArtistsForAlbum(ctx, db, albumRowID)
	if err != nil {
		return nil, err
	}

	tracks, err := fetchTracksForAlbum(ctx, db, albumRowID)
	if err != nil {
		return nil, err
	}

	return &AlbumDetail{
		AlbumSummary: AlbumSummary{
			ID:          albumID,
			Name:        name,
			AlbumType:   albumType,
			ReleaseDate: releaseDate,
			TotalTracks: albumTotalTracks,
			Popularity:  albumPop,
			Images:      images,
		},
		Label:                label,
		ReleaseDatePrecision: releasePrecision,
		ExternalUPC:          nullStringPtr(externalUPC),
		CopyrightC:           nullStringPtr(copyrightC),
		CopyrightP:           nullStringPtr(copyrightP),
		Tracks:               tracks,
		Artists:              artists,
	}, nil
}

func queryArtistSearch(ctx context.Context, db *sql.DB, params ArtistSearchParams) ([]ArtistSummary, error) {
	where := []string{"1=1"}
	args := []any{}
	if params.Query != "" {
		where = append(where, "ar.name LIKE ?")
		args = append(args, formatQueryLike(params.Query))
	}
	if params.Genre != "" {
		where = append(where, "EXISTS (SELECT 1 FROM artist_genres ag WHERE ag.artist_rowid = ar.rowid AND ag.genre LIKE ?)")
		args = append(args, formatQueryLike(params.Genre))
	}

	query := fmt.Sprintf(`
SELECT
	 ar.rowid,
	 ar.id,
	 ar.name,
	 ar.popularity,
	 ar.followers_total
FROM artists ar
WHERE %s
ORDER BY ar.popularity DESC
LIMIT ? OFFSET ?`, strings.Join(where, " AND "))

	args = append(args, params.Limit, params.Offset)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		artistRowIDs []int64
		results      []ArtistSummary
	)
	for rows.Next() {
		var (
			rowID      int64
			id         string
			name       string
			popularity int
			followers  int
		)
		if err := rows.Scan(&rowID, &id, &name, &popularity, &followers); err != nil {
			return nil, err
		}
		artistRowIDs = append(artistRowIDs, rowID)
		results = append(results, ArtistSummary{
			ID:             id,
			Name:           name,
			Popularity:     popularity,
			FollowersTotal: followers,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return results, nil
	}

	genresByArtist, err := fetchArtistGenres(ctx, db, artistRowIDs)
	if err != nil {
		return nil, err
	}
	imagesByArtist, err := fetchArtistImages(ctx, db, artistRowIDs)
	if err != nil {
		return nil, err
	}

	for i := range results {
		results[i].Genres = genresByArtist[artistRowIDs[i]]
		results[i].Images = imagesByArtist[artistRowIDs[i]]
	}

	return results, nil
}

func queryArtistDetail(ctx context.Context, db *sql.DB, artistID string) (*ArtistDetail, error) {
	var (
		artistRowID int64
		name        string
		popularity  int
		followers   int
	)

	query := `
SELECT rowid, id, name, popularity, followers_total
FROM artists
WHERE id = ?`

	if err := db.QueryRowContext(ctx, query, artistID).Scan(&artistRowID, &artistID, &name, &popularity, &followers); err != nil {
		return nil, err
	}

	genres, err := fetchArtistGenres(ctx, db, []int64{artistRowID})
	if err != nil {
		return nil, err
	}
	images, err := fetchArtistImages(ctx, db, []int64{artistRowID})
	if err != nil {
		return nil, err
	}
	albums, err := fetchArtistAlbums(ctx, db, artistRowID)
	if err != nil {
		return nil, err
	}

	return &ArtistDetail{
		ArtistSummary: ArtistSummary{
			ID:             artistID,
			Name:           name,
			Popularity:     popularity,
			FollowersTotal: followers,
			Genres:         genres[artistRowID],
			Images:         images[artistRowID],
		},
		Albums: albums,
	}, nil
}

func queryPlaylistDetail(ctx context.Context, db *sql.DB, playlistID string) (*PlaylistDetail, error) {
	return queryPlaylistDetailWithParams(ctx, db, playlistID, PlaylistParams{Limit: 200, Offset: 0})
}

func queryPlaylistDetailWithParams(ctx context.Context, db *sql.DB, playlistID string, params PlaylistParams) (*PlaylistDetail, error) {
	var (
		rowID       int64
		snapshotID  string
		fetchedAt   int64
		name        string
		description sql.NullString
		ownerID     sql.NullString
		ownerName   sql.NullString
		followers   sql.NullInt64
		tracksTotal int
	)

	query := `
SELECT rowid, id, snapshot_id, fetched_at, name, description, owner_id, owner_display_name, followers_total, tracks_total
FROM playlists.playlists
WHERE id = ?`

	if err := db.QueryRowContext(ctx, query, playlistID).Scan(
		&rowID,
		&playlistID,
		&snapshotID,
		&fetchedAt,
		&name,
		&description,
		&ownerID,
		&ownerName,
		&followers,
		&tracksTotal,
	); err != nil {
		return nil, err
	}

	images, err := fetchPlaylistImages(ctx, db, rowID)
	if err != nil {
		return nil, err
	}

	tracks, err := fetchPlaylistTracks(ctx, db, rowID, params)
	if err != nil {
		return nil, err
	}

	return &PlaylistDetail{
		ID:             playlistID,
		SnapshotID:     snapshotID,
		Name:           name,
		Description:    nullStringPtr(description),
		OwnerID:        nullStringPtr(ownerID),
		OwnerName:      nullStringPtr(ownerName),
		FollowersTotal: nullIntPtr(followers),
		TracksTotal:    tracksTotal,
		FetchedAt:      unixStringPtr(fetchedAt),
		Images:         images,
		Tracks:         tracks,
	}, nil
}

func fetchTrackArtists(ctx context.Context, db *sql.DB, trackRowIDs []int64) (map[int64][]ArtistSummary, error) {
	if len(trackRowIDs) == 0 {
		return map[int64][]ArtistSummary{}, nil
	}

	placeholders := buildInClause(len(trackRowIDs))
	args := make([]any, len(trackRowIDs))
	for i, id := range trackRowIDs {
		args[i] = id
	}

	query := fmt.Sprintf(`
SELECT ta.track_rowid, ar.id, ar.name, ar.popularity, ar.followers_total
FROM track_artists ta
JOIN artists ar ON ar.rowid = ta.artist_rowid
WHERE ta.track_rowid IN (%s)
ORDER BY ar.popularity DESC`, placeholders)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64][]ArtistSummary)
	for rows.Next() {
		var (
			trackRowID int64
			artistID   string
			name       string
			popularity int
			followers  int
		)
		if err := rows.Scan(&trackRowID, &artistID, &name, &popularity, &followers); err != nil {
			return nil, err
		}
		result[trackRowID] = append(result[trackRowID], ArtistSummary{
			ID:             artistID,
			Name:           name,
			Popularity:     popularity,
			FollowersTotal: followers,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func fetchArtistsForTrack(ctx context.Context, db *sql.DB, trackRowID int64) ([]ArtistDetail, error) {
	query := `
SELECT ar.rowid, ar.id, ar.name, ar.popularity, ar.followers_total
FROM track_artists ta
JOIN artists ar ON ar.rowid = ta.artist_rowid
WHERE ta.track_rowid = ?
ORDER BY ar.popularity DESC`

	rows, err := db.QueryContext(ctx, query, trackRowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		artistRowIDs []int64
		artists      []ArtistDetail
	)
	for rows.Next() {
		var (
			rowID      int64
			id         string
			name       string
			popularity int
			followers  int
		)
		if err := rows.Scan(&rowID, &id, &name, &popularity, &followers); err != nil {
			return nil, err
		}
		artistRowIDs = append(artistRowIDs, rowID)
		artists = append(artists, ArtistDetail{
			ArtistSummary: ArtistSummary{
				ID:             id,
				Name:           name,
				Popularity:     popularity,
				FollowersTotal: followers,
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(artists) == 0 {
		return artists, nil
	}

	genresByArtist, err := fetchArtistGenres(ctx, db, artistRowIDs)
	if err != nil {
		return nil, err
	}
	imagesByArtist, err := fetchArtistImages(ctx, db, artistRowIDs)
	if err != nil {
		return nil, err
	}

	for i := range artists {
		rowID := artistRowIDs[i]
		artists[i].Genres = genresByArtist[rowID]
		artists[i].Images = imagesByArtist[rowID]
	}

	return artists, nil
}

func fetchArtistsForAlbum(ctx context.Context, db *sql.DB, albumRowID int64) ([]ArtistSummary, error) {
	query := `
SELECT ar.id, ar.name, ar.popularity, ar.followers_total
FROM artist_albums aa
JOIN artists ar ON ar.rowid = aa.artist_rowid
WHERE aa.album_rowid = ?
ORDER BY ar.popularity DESC`

	rows, err := db.QueryContext(ctx, query, albumRowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artists []ArtistSummary
	for rows.Next() {
		var (
			id         string
			name       string
			popularity int
			followers  int
		)
		if err := rows.Scan(&id, &name, &popularity, &followers); err != nil {
			return nil, err
		}
		artists = append(artists, ArtistSummary{
			ID:             id,
			Name:           name,
			Popularity:     popularity,
			FollowersTotal: followers,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return artists, nil
}

func fetchTracksForAlbum(ctx context.Context, db *sql.DB, albumRowID int64) ([]TrackSummary, error) {
	query := `
SELECT rowid, id, name, popularity, duration_ms, explicit, preview_url, external_id_isrc
FROM tracks
WHERE album_rowid = ?
ORDER BY disc_number ASC, track_number ASC`

	rows, err := db.QueryContext(ctx, query, albumRowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		trackRowIDs []int64
		tracks      []TrackSummary
	)

	for rows.Next() {
		var (
			rowID        int64
			trackID      string
			name         string
			popularity   int
			durationMS   int
			explicit     int64
			previewURL   sql.NullString
			externalISRC sql.NullString
		)
		if err := rows.Scan(&rowID, &trackID, &name, &popularity, &durationMS, &explicit, &previewURL, &externalISRC); err != nil {
			return nil, err
		}
		trackRowIDs = append(trackRowIDs, rowID)
		tracks = append(tracks, TrackSummary{
			ID:           trackID,
			Name:         name,
			Popularity:   popularity,
			DurationMS:   durationMS,
			Explicit:     parseBoolInt(explicit),
			PreviewURL:   nullStringPtr(previewURL),
			ExternalISRC: nullStringPtr(externalISRC),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(tracks) == 0 {
		return tracks, nil
	}

	artistsByTrack, err := fetchTrackArtists(ctx, db, trackRowIDs)
	if err != nil {
		return nil, err
	}

	for i := range tracks {
		rowID := trackRowIDs[i]
		tracks[i].Artists = artistsByTrack[rowID]
	}

	return tracks, nil
}

func fetchArtistAlbums(ctx context.Context, db *sql.DB, artistRowID int64) ([]ArtistAlbumSummary, error) {
	query := `
SELECT
	 al.rowid,
	 al.id,
	 al.name,
	 al.album_type,
	 al.release_date,
	 al.total_tracks,
	 al.popularity,
	 aa.is_appears_on,
	 aa.is_implicit_appears_on,
	 aa.index_in_album
FROM artist_albums aa
JOIN albums al ON al.rowid = aa.album_rowid
WHERE aa.artist_rowid = ?
ORDER BY al.release_date DESC`

	rows, err := db.QueryContext(ctx, query, artistRowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		albums      []ArtistAlbumSummary
		albumRowIDs []int64
	)
	for rows.Next() {
		var (
			albumRowID          int64
			albumID             string
			name                string
			albumType           string
			releaseDate         string
			totalTracks         int
			popularity          int
			isAppearsOn         int64
			isImplicitAppearsOn int64
			indexInAlbum        sql.NullInt64
		)
		if err := rows.Scan(&albumRowID, &albumID, &name, &albumType, &releaseDate, &totalTracks, &popularity, &isAppearsOn, &isImplicitAppearsOn, &indexInAlbum); err != nil {
			return nil, err
		}
		albumRowIDs = append(albumRowIDs, albumRowID)
		albums = append(albums, ArtistAlbumSummary{
			AlbumSummary: AlbumSummary{
				ID:          albumID,
				Name:        name,
				AlbumType:   albumType,
				ReleaseDate: releaseDate,
				TotalTracks: totalTracks,
				Popularity:  popularity,
			},
			IsAppearsOn:         parseBoolInt(isAppearsOn),
			IsImplicitAppearsOn: parseBoolInt(isImplicitAppearsOn),
			IndexInAlbum:        nullIntPtr(indexInAlbum),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(albums) == 0 {
		return albums, nil
	}

	imagesByAlbum, err := fetchAlbumImagesBatch(ctx, db, albumRowIDs)
	if err != nil {
		return nil, err
	}

	for i := range albums {
		albums[i].Images = imagesByAlbum[albumRowIDs[i]]
	}

	return albums, nil
}

func fetchAlbumImages(ctx context.Context, db *sql.DB, albumRowID int64) ([]Image, error) {
	query := `
SELECT width, height, url
FROM album_images
WHERE album_rowid = ?
ORDER BY width DESC`

	rows, err := db.QueryContext(ctx, query, albumRowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []Image
	for rows.Next() {
		var img Image
		if err := rows.Scan(&img.Width, &img.Height, &img.URL); err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return images, nil
}

func fetchAlbumImagesBatch(ctx context.Context, db *sql.DB, albumRowIDs []int64) (map[int64][]Image, error) {
	if len(albumRowIDs) == 0 {
		return map[int64][]Image{}, nil
	}

	placeholders := buildInClause(len(albumRowIDs))
	args := make([]any, len(albumRowIDs))
	for i, id := range albumRowIDs {
		args[i] = id
	}

	query := fmt.Sprintf(`
SELECT album_rowid, width, height, url
FROM album_images
WHERE album_rowid IN (%s)
ORDER BY width DESC`, placeholders)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64][]Image)
	for rows.Next() {
		var (
			albumRowID int64
			img        Image
		)
		if err := rows.Scan(&albumRowID, &img.Width, &img.Height, &img.URL); err != nil {
			return nil, err
		}
		result[albumRowID] = append(result[albumRowID], img)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func fetchArtistGenres(ctx context.Context, db *sql.DB, artistRowIDs []int64) (map[int64][]string, error) {
	if len(artistRowIDs) == 0 {
		return map[int64][]string{}, nil
	}

	placeholders := buildInClause(len(artistRowIDs))
	args := make([]any, len(artistRowIDs))
	for i, id := range artistRowIDs {
		args[i] = id
	}

	query := fmt.Sprintf(`
SELECT artist_rowid, genre
FROM artist_genres
WHERE artist_rowid IN (%s)`, placeholders)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64][]string)
	for rows.Next() {
		var (
			artistRowID int64
			genre       string
		)
		if err := rows.Scan(&artistRowID, &genre); err != nil {
			return nil, err
		}
		result[artistRowID] = append(result[artistRowID], genre)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for key := range result {
		sort.Strings(result[key])
	}

	return result, nil
}

func fetchArtistImages(ctx context.Context, db *sql.DB, artistRowIDs []int64) (map[int64][]Image, error) {
	if len(artistRowIDs) == 0 {
		return map[int64][]Image{}, nil
	}

	placeholders := buildInClause(len(artistRowIDs))
	args := make([]any, len(artistRowIDs))
	for i, id := range artistRowIDs {
		args[i] = id
	}

	query := fmt.Sprintf(`
SELECT artist_rowid, width, height, url
FROM artist_images
WHERE artist_rowid IN (%s)
ORDER BY width DESC`, placeholders)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64][]Image)
	for rows.Next() {
		var (
			artistRowID int64
			img         Image
		)
		if err := rows.Scan(&artistRowID, &img.Width, &img.Height, &img.URL); err != nil {
			return nil, err
		}
		result[artistRowID] = append(result[artistRowID], img)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func fetchTrackFile(ctx context.Context, db *sql.DB, trackID string) (*TrackFileDetail, error) {
	query := `
SELECT status, filename, filesize_bytes, sha256_original, reencoded_kbit_vbr, fetched_at, session_country, content_ratings,
	 file_id_ogg_vorbis_96, file_id_ogg_vorbis_160, file_id_ogg_vorbis_320, file_id_aac_24, file_id_mp3_96, alternatives
FROM track_urls.track_files
WHERE track_id = ?`

	var (
		status        string
		filename      sql.NullString
		filesizeBytes sql.NullInt64
		sha256        sql.NullString
		reencoded     sql.NullInt64
		fetchedAt     sql.NullInt64
		session       sql.NullString
		content       sql.NullString
		ogg96         sql.NullString
		ogg160        sql.NullString
		ogg320        sql.NullString
		aac24         sql.NullString
		mp396         sql.NullString
		alternatives  sql.NullString
	)

	err := db.QueryRowContext(ctx, query, trackID).Scan(
		&status,
		&filename,
		&filesizeBytes,
		&sha256,
		&reencoded,
		&fetchedAt,
		&session,
		&content,
		&ogg96,
		&ogg160,
		&ogg320,
		&aac24,
		&mp396,
		&alternatives,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &TrackFileDetail{
		Status:             status,
		Filename:           nullStringPtr(filename),
		FilesizeBytes:      nullInt64Ptr(filesizeBytes),
		SHA256Original:     nullStringPtr(sha256),
		ReencodedKbitVBR:   nullInt64Ptr(reencoded),
		FetchedAt:          unixStringPtrNullable(fetchedAt),
		SessionCountry:     nullStringPtr(session),
		ContentRatings:     nullStringPtr(content),
		FileIDOGGVorbis96:  nullStringPtr(ogg96),
		FileIDOGGVorbis160: nullStringPtr(ogg160),
		FileIDOGGVorbis320: nullStringPtr(ogg320),
		FileIDAAC24:        nullStringPtr(aac24),
		FileIDMP396:        nullStringPtr(mp396),
		Alternatives:       parseJSONList(alternatives.String),
	}, nil
}

func fetchPlaylistImages(ctx context.Context, db *sql.DB, playlistRowID int64) ([]PlaylistImage, error) {
	query := `
SELECT url
FROM playlists.playlist_images
WHERE playlist_rowid = ?`

	rows, err := db.QueryContext(ctx, query, playlistRowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []PlaylistImage
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return nil, err
		}
		images = append(images, PlaylistImage{URL: url})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return images, nil
}

func fetchPlaylistTracks(ctx context.Context, db *sql.DB, playlistRowID int64, params PlaylistParams) ([]PlaylistTrack, error) {
	query := `
SELECT
	 pt.position,
	 pt.added_at,
	 pt.added_by_id,
	 pt.is_episode,
	 pt.is_local,
	 pt.track_rowid,
	 pt.name_if_is_local,
	 pt.uri_if_is_local,
	 pt.album_name_if_is_local,
	 pt.artists_name_if_is_local,
	 pt.duration_ms_if_is_local,
	 t.id,
	 t.name,
	 t.duration_ms,
	 t.explicit,
	 t.popularity,
	 t.preview_url,
	 t.external_id_isrc,
	 a.id,
	 a.name,
	 a.album_type,
	 a.release_date,
	 a.total_tracks,
	 a.popularity
FROM playlists.playlist_tracks pt
LEFT JOIN tracks t ON t.rowid = pt.track_rowid
LEFT JOIN albums a ON a.rowid = t.album_rowid
WHERE pt.playlist_rowid = ?
ORDER BY pt.position ASC
LIMIT ? OFFSET ?`

	rows, err := db.QueryContext(ctx, query, playlistRowID, params.Limit, params.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		playlistTracks []PlaylistTrack
		trackRowIDs    []int64
		trackRowIndex  []int
	)

	for rows.Next() {
		var (
			position        int
			addedAt         sql.NullInt64
			addedBy         sql.NullString
			isEpisode       int64
			isLocal         int64
			trackRowID      sql.NullInt64
			localName       sql.NullString
			localURI        sql.NullString
			localAlbum      sql.NullString
			localArtists    sql.NullString
			localDuration   sql.NullInt64
			trackID         sql.NullString
			trackName       sql.NullString
			trackDuration   sql.NullInt64
			trackExplicit   sql.NullInt64
			trackPopularity sql.NullInt64
			trackPreview    sql.NullString
			trackISRC       sql.NullString
			albumID         sql.NullString
			albumName       sql.NullString
			albumType       sql.NullString
			releaseDate     sql.NullString
			albumTotal      sql.NullInt64
			albumPop        sql.NullInt64
		)

		if err := rows.Scan(
			&position,
			&addedAt,
			&addedBy,
			&isEpisode,
			&isLocal,
			&trackRowID,
			&localName,
			&localURI,
			&localAlbum,
			&localArtists,
			&localDuration,
			&trackID,
			&trackName,
			&trackDuration,
			&trackExplicit,
			&trackPopularity,
			&trackPreview,
			&trackISRC,
			&albumID,
			&albumName,
			&albumType,
			&releaseDate,
			&albumTotal,
			&albumPop,
		); err != nil {
			return nil, err
		}

		var trackSummary *TrackSummary
		if trackID.Valid {
			trackSummary = &TrackSummary{
				ID:           trackID.String,
				Name:         trackName.String,
				Popularity:   intFromNull(trackPopularity),
				DurationMS:   intFromNull(trackDuration),
				Explicit:     parseBoolInt(int64FromNull(trackExplicit)),
				PreviewURL:   nullStringPtr(trackPreview),
				ExternalISRC: nullStringPtr(trackISRC),
				Album: AlbumSummary{
					ID:          albumID.String,
					Name:        albumName.String,
					AlbumType:   albumType.String,
					ReleaseDate: releaseDate.String,
					TotalTracks: intFromNull(albumTotal),
					Popularity:  intFromNull(albumPop),
				},
			}
			if trackRowID.Valid {
				trackRowIDs = append(trackRowIDs, trackRowID.Int64)
				trackRowIndex = append(trackRowIndex, len(playlistTracks))
			}
		}

		var localDetail *LocalTrack
		if isLocal != 0 {
			localDetail = &LocalTrack{
				Name:        nullStringPtr(localName),
				URI:         nullStringPtr(localURI),
				AlbumName:   nullStringPtr(localAlbum),
				ArtistsName: nullStringPtr(localArtists),
				DurationMS:  intPtrFromNull(localDuration),
			}
		}

		playlistTracks = append(playlistTracks, PlaylistTrack{
			Position:         position,
			AddedAt:          unixStringPtrNullable(addedAt),
			AddedByID:        nullStringPtr(addedBy),
			IsEpisode:        parseBoolInt(isEpisode),
			IsLocal:          parseBoolInt(isLocal),
			Track:            trackSummary,
			LocalTrackDetail: localDetail,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(trackRowIDs) == 0 {
		return playlistTracks, nil
	}

	artistsByTrack, err := fetchTrackArtists(ctx, db, trackRowIDs)
	if err != nil {
		return nil, err
	}

	for i, trackRowID := range trackRowIDs {
		trackIndex := trackRowIndex[i]
		track := playlistTracks[trackIndex].Track
		if track != nil {
			track.Artists = artistsByTrack[trackRowID]
		}
	}

	return playlistTracks, nil
}

func buildInClause(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	v := value.String
	return &v
}

func nullInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	v := value.Int64
	return &v
}

func nullIntPtr(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	v := int(value.Int64)
	return &v
}

func intFromNull(value sql.NullInt64) int {
	if !value.Valid {
		return 0
	}
	return int(value.Int64)
}

func int64FromNull(value sql.NullInt64) int64 {
	if !value.Valid {
		return 0
	}
	return value.Int64
}

func intPtrFromNull(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	v := int(value.Int64)
	return &v
}

func unixStringPtr(value int64) *string {
	if value == 0 {
		return nil
	}
	var t time.Time
	if value > 1e12 {
		t = time.UnixMilli(value).UTC()
	} else {
		t = time.Unix(value, 0).UTC()
	}
	formatted := t.Format(time.RFC3339)
	return &formatted
}

func unixStringPtrNullable(value sql.NullInt64) *string {
	if !value.Valid {
		return nil
	}
	return unixStringPtr(value.Int64)
}
