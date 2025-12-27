package api

type TrackSummary struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Popularity   int             `json:"popularity"`
	DurationMS   int             `json:"duration_ms"`
	Explicit     bool            `json:"explicit"`
	PreviewURL   *string         `json:"preview_url"`
	Album        AlbumSummary    `json:"album,omitempty"`
	Artists      []ArtistSummary `json:"artists,omitempty"`
	ExternalISRC *string         `json:"external_isrc,omitempty"`
}

type TrackDetail struct {
	Track     TrackSummary     `json:"track"`
	Album     AlbumDetail      `json:"album"`
	Artists   []ArtistDetail   `json:"artists"`
	TrackFile *TrackFileDetail `json:"track_file,omitempty"`
	FetchedAt *string          `json:"fetched_at,omitempty"`
}

type TrackFileDetail struct {
	Status             string   `json:"status"`
	Filename           *string  `json:"filename"`
	FilesizeBytes      *int64   `json:"filesize_bytes"`
	SHA256Original     *string  `json:"sha256_original"`
	ReencodedKbitVBR   *int64   `json:"reencoded_kbit_vbr"`
	FetchedAt          *string  `json:"fetched_at"`
	SessionCountry     *string  `json:"session_country"`
	ContentRatings     *string  `json:"content_ratings"`
	FileIDOGGVorbis96  *string  `json:"file_id_ogg_vorbis_96"`
	FileIDOGGVorbis160 *string  `json:"file_id_ogg_vorbis_160"`
	FileIDOGGVorbis320 *string  `json:"file_id_ogg_vorbis_320"`
	FileIDAAC24        *string  `json:"file_id_aac_24"`
	FileIDMP396        *string  `json:"file_id_mp3_96"`
	Alternatives       []string `json:"alternatives,omitempty"`
}

type AlbumSummary struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	AlbumType   string  `json:"album_type"`
	ReleaseDate string  `json:"release_date"`
	TotalTracks int     `json:"total_tracks"`
	Popularity  int     `json:"popularity"`
	Images      []Image `json:"images,omitempty"`
}

type AlbumDetail struct {
	AlbumSummary
	Label                string          `json:"label"`
	ReleaseDatePrecision string          `json:"release_date_precision"`
	ExternalUPC          *string         `json:"external_upc"`
	CopyrightC           *string         `json:"copyright_c"`
	CopyrightP           *string         `json:"copyright_p"`
	Tracks               []TrackSummary  `json:"tracks,omitempty"`
	Artists              []ArtistSummary `json:"artists,omitempty"`
}

type ArtistSummary struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Popularity     int      `json:"popularity"`
	FollowersTotal int      `json:"followers_total"`
	Genres         []string `json:"genres,omitempty"`
	Images         []Image  `json:"images,omitempty"`
}

type ArtistDetail struct {
	ArtistSummary
	Albums []ArtistAlbumSummary `json:"albums,omitempty"`
}

type ArtistAlbumSummary struct {
	AlbumSummary
	IsAppearsOn         bool `json:"is_appears_on"`
	IsImplicitAppearsOn bool `json:"is_implicit_appears_on"`
	IndexInAlbum        *int `json:"index_in_album"`
}

type PlaylistDetail struct {
	ID             string          `json:"id"`
	SnapshotID     string          `json:"snapshot_id"`
	Name           string          `json:"name"`
	Description    *string         `json:"description"`
	OwnerID        *string         `json:"owner_id"`
	OwnerName      *string         `json:"owner_display_name"`
	FollowersTotal *int            `json:"followers_total"`
	TracksTotal    int             `json:"tracks_total"`
	FetchedAt      *string         `json:"fetched_at"`
	Images         []PlaylistImage `json:"images,omitempty"`
	Tracks         []PlaylistTrack `json:"tracks,omitempty"`
}

type PlaylistImage struct {
	URL string `json:"url"`
}

type PlaylistTrack struct {
	Position         int           `json:"position"`
	AddedAt          *string       `json:"added_at"`
	AddedByID        *string       `json:"added_by_id"`
	IsEpisode        bool          `json:"is_episode"`
	IsLocal          bool          `json:"is_local"`
	Track            *TrackSummary `json:"track,omitempty"`
	LocalTrackDetail *LocalTrack   `json:"local_track,omitempty"`
}

type LocalTrack struct {
	Name        *string `json:"name"`
	URI         *string `json:"uri"`
	AlbumName   *string `json:"album_name"`
	ArtistsName *string `json:"artists_name"`
	DurationMS  *int    `json:"duration_ms"`
}

type Image struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	URL    string `json:"url"`
}
