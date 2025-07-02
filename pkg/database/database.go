package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// DB represents the database connection
type DB struct {
	conn *sql.DB
}

// New creates a new database connection
func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection parameters
	conn.SetMaxOpenConns(1) // SQLite only supports one writer at a time
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(time.Hour)

	db := &DB{conn: conn}

	// Run migrations
	migrations := []Migration{
		InitialMigration(),
		// Add new migrations here
	}

	if err := db.RunMigrations(migrations); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to run database migrations: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// GetRecentlyWatchedAnime returns recently watched anime from the database
func (db *DB) GetRecentlyWatchedAnime(limit int) ([]*Anime, error) {
	rows, err := db.conn.Query(`
		SELECT DISTINCT a.id, a.title, a.original_title, a.alternative_titles, a.description, 
		       a.total_episodes, a.type, a.year, a.season, a.status, a.genres, 
		       a.thumbnail_url, a.created_at, a.updated_at
		FROM anime a
		JOIN episode_progress ep ON a.id = ep.anime_id
		ORDER BY ep.last_watched DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recently watched anime: %w", err)
	}
	defer rows.Close()

	var animes []*Anime
	for rows.Next() {
		var anime Anime
		var alternativeTitlesJSON, genresJSON []byte

		err := rows.Scan(
			&anime.ID, &anime.Title, &anime.OriginalTitle, &alternativeTitlesJSON,
			&anime.Description, &anime.TotalEpisodes, &anime.Type, &anime.Year,
			&anime.Season, &anime.Status, &genresJSON, &anime.ThumbnailURL,
			&anime.CreatedAt, &anime.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan anime: %w", err)
		}

		// Parse JSON fields
		if err := json.Unmarshal(alternativeTitlesJSON, &anime.AlternativeTitles); err != nil {
			return nil, fmt.Errorf("failed to parse alternative titles: %w", err)
		}

		if err := json.Unmarshal(genresJSON, &anime.Genres); err != nil {
			return nil, fmt.Errorf("failed to parse genres: %w", err)
		}

		animes = append(animes, &anime)
	}

	return animes, rows.Err()
}

// GetCurrentlyWatchingAnime returns anime that the user is currently watching
func (db *DB) GetCurrentlyWatchingAnime() ([]*Anime, error) {
	rows, err := db.conn.Query(`
		SELECT DISTINCT a.id, a.title, a.original_title, a.alternative_titles, a.description, 
		       a.total_episodes, a.type, a.year, a.season, a.status, a.genres, 
		       a.thumbnail_url, a.created_at, a.updated_at
		FROM anime a
		JOIN anime_tracking at ON a.id = at.anime_id
		WHERE at.status = 'watching'
		ORDER BY at.last_updated DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query currently watching anime: %w", err)
	}
	defer rows.Close()

	var animes []*Anime
	for rows.Next() {
		var anime Anime
		var alternativeTitlesJSON, genresJSON []byte

		err := rows.Scan(
			&anime.ID, &anime.Title, &anime.OriginalTitle, &alternativeTitlesJSON,
			&anime.Description, &anime.TotalEpisodes, &anime.Type, &anime.Year,
			&anime.Season, &anime.Status, &genresJSON, &anime.ThumbnailURL,
			&anime.CreatedAt, &anime.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan anime: %w", err)
		}

		// Parse JSON fields
		if err := json.Unmarshal(alternativeTitlesJSON, &anime.AlternativeTitles); err != nil {
			return nil, fmt.Errorf("failed to parse alternative titles: %w", err)
		}

		if err := json.Unmarshal(genresJSON, &anime.Genres); err != nil {
			return nil, fmt.Errorf("failed to parse genres: %w", err)
		}

		animes = append(animes, &anime)
	}

	return animes, rows.Err()
}

// GetAnimeByStatus returns all anime with the given status
func (db *DB) GetAnimeByStatus(status string) ([]Anime, error) {
	var animes []Anime
	rows, err := db.conn.Query(`
		SELECT id, title, original_title, alternative_titles, description, 
		       total_episodes, type, year, season, status, genres, thumbnail_url
		FROM anime
		WHERE status = ?
	`, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query anime by status: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var anime Anime
		var altTitlesJSON, genresJSON string
		err := rows.Scan(
			&anime.ID,
			&anime.Title,
			&anime.OriginalTitle,
			&altTitlesJSON,
			&anime.Description,
			&anime.TotalEpisodes,
			&anime.Type,
			&anime.Year,
			&anime.Season,
			&anime.Status,
			&genresJSON,
			&anime.ThumbnailURL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan anime row: %w", err)
		}

		// Parse alternative titles JSON
		if err := json.Unmarshal([]byte(altTitlesJSON), &anime.AlternativeTitles); err != nil {
			return nil, fmt.Errorf("failed to parse alternative titles: %w", err)
		}

		// Parse genres JSON
		if err := json.Unmarshal([]byte(genresJSON), &anime.Genres); err != nil {
			return nil, fmt.Errorf("failed to parse genres: %w", err)
		}

		animes = append(animes, anime)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating anime rows: %w", err)
	}

	return animes, nil
}
