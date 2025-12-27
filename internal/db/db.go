package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	DB *sql.DB
}

func Open(ctx context.Context, metadataPath, playlistsPath, trackURLsPath string) (*Store, error) {
	if metadataPath == "" {
		return nil, fmt.Errorf("metadata database path is required")
	}

	db, err := sql.Open("sqlite3", fileDSN(metadataPath))
	if err != nil {
		return nil, err
	}
	// Keep a single connection so ATTACH applies to all queries.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	if playlistsPath != "" {
		if err := attachDB(ctx, db, "playlists", playlistsPath); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("attach playlists: %w", err)
		}
	}

	if trackURLsPath != "" {
		if err := attachDB(ctx, db, "track_urls", trackURLsPath); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("attach track urls: %w", err)
		}
	}

	return &Store{DB: db}, nil
}

func (s *Store) Close() error {
	return s.DB.Close()
}

func attachDB(ctx context.Context, db *sql.DB, alias, path string) error {
	dsn := fileDSN(path)
	_, err := db.ExecContext(ctx, fmt.Sprintf("ATTACH DATABASE ? AS %s", alias), dsn)
	return err
}

func fileDSN(path string) string {
	q := url.Values{}
	q.Add("mode", "ro")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(1)")

	u := &url.URL{Scheme: "file", Path: path, RawQuery: q.Encode()}
	return u.String()
}
