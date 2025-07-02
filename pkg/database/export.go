package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// BackupData represents the structure of a database backup
type BackupData struct {
	Version         int               `json:"version"`
	CreatedAt       time.Time         `json:"created_at"`
	Config          []ConfigEntry     `json:"config"`
	Anime           []Anime           `json:"anime"`
	AnimeTracking   []AnimeTracking   `json:"anime_tracking"`
	EpisodeProgress []EpisodeProgress `json:"episode_progress"`
	Episodes        []Episode         `json:"episodes"`
	Extensions      []Extension       `json:"extensions"`
	Sources         []Source          `json:"sources"`
	AnimeSources    []AnimeSource     `json:"anime_sources"`
}

// ExportToJSON exports the database to a JSON file
func (db *DB) ExportToJSON(filePath string) error {
	var err error
	data := BackupData{
		Version:   1,
		CreatedAt: time.Now(),
	}

	// Get all config entries
	data.Config, err = db.GetAllConfig()
	if err != nil {
		return fmt.Errorf("failed to export config: %w", err)
	}

	// Get all anime
	rows, err := db.conn.Query(`
		SELECT 
			id, title, original_title, alternative_titles, description, 
			total_episodes, type, year, season, status, genres, thumbnail_url,
			created_at, updated_at
		FROM anime
	`)
	if err != nil {
		return fmt.Errorf("failed to query anime: %w", err)
	}
	defer rows.Close()

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
			return fmt.Errorf("failed to scan anime: %w", err)
		}

		// Parse JSON fields
		if err := json.Unmarshal(alternativeTitlesJSON, &anime.AlternativeTitles); err != nil {
			return fmt.Errorf("failed to unmarshal alternative titles: %w", err)
		}

		if err := json.Unmarshal(genresJSON, &anime.Genres); err != nil {
			return fmt.Errorf("failed to unmarshal genres: %w", err)
		}

		data.Anime = append(data.Anime, anime)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating anime rows: %w", err)
	}

	// Get all anime tracking entries
	rows, err = db.conn.Query(`
		SELECT 
			id, anime_id, tracker, tracker_id, status, score, 
			current_episode, total_episodes, last_updated
		FROM anime_tracking
	`)
	if err != nil {
		return fmt.Errorf("failed to query anime tracking: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tracking AnimeTracking
		err := rows.Scan(
			&tracking.ID, &tracking.AnimeID, &tracking.Tracker, &tracking.TrackerID,
			&tracking.Status, &tracking.Score, &tracking.CurrentEpisode,
			&tracking.TotalEpisodes, &tracking.LastUpdated,
		)
		if err != nil {
			return fmt.Errorf("failed to scan anime tracking: %w", err)
		}

		data.AnimeTracking = append(data.AnimeTracking, tracking)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating anime tracking rows: %w", err)
	}

	// Get all episode progress entries
	rows, err = db.conn.Query(`
		SELECT 
			id, anime_id, episode_number, position, duration, 
			playback_speed, watched, source_id, last_watched
		FROM episode_progress
	`)
	if err != nil {
		return fmt.Errorf("failed to query episode progress: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var progress EpisodeProgress
		err := rows.Scan(
			&progress.ID, &progress.AnimeID, &progress.EpisodeNumber, &progress.Position,
			&progress.Duration, &progress.PlaybackSpeed, &progress.Watched,
			&progress.SourceID, &progress.LastWatched,
		)
		if err != nil {
			return fmt.Errorf("failed to scan episode progress: %w", err)
		}

		data.EpisodeProgress = append(data.EpisodeProgress, progress)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating episode progress rows: %w", err)
	}

	// Get all episodes
	rows, err = db.conn.Query(`
		SELECT 
			id, anime_id, number, title, description, duration, 
			thumbnail_url, air_date, is_filler, created_at
		FROM episode
	`)
	if err != nil {
		return fmt.Errorf("failed to query episodes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var episode Episode
		err := rows.Scan(
			&episode.ID, &episode.AnimeID, &episode.Number, &episode.Title,
			&episode.Description, &episode.Duration, &episode.ThumbnailURL,
			&episode.AirDate, &episode.IsFiller, &episode.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to scan episode: %w", err)
		}

		data.Episodes = append(data.Episodes, episode)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating episode rows: %w", err)
	}

	// Get all extensions
	rows, err = db.conn.Query(`
		SELECT 
			id, name, package, language, version, nsfw, path, repository_url,
			installed_at, updated_at
		FROM extension
	`)
	if err != nil {
		return fmt.Errorf("failed to query extensions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ext Extension
		err := rows.Scan(
			&ext.ID, &ext.Name, &ext.Package, &ext.Language, &ext.Version,
			&ext.NSFW, &ext.Path, &ext.RepositoryURL, &ext.InstalledAt, &ext.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to scan extension: %w", err)
		}

		data.Extensions = append(data.Extensions, ext)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating extension rows: %w", err)
	}

	// Get all sources
	rows, err = db.conn.Query(`
		SELECT 
			id, source_id, extension_id, name, language, base_url, nsfw
		FROM source
	`)
	if err != nil {
		return fmt.Errorf("failed to query sources: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var source Source
		err := rows.Scan(
			&source.ID, &source.SourceID, &source.ExtensionID, &source.Name,
			&source.Language, &source.BaseURL, &source.NSFW,
		)
		if err != nil {
			return fmt.Errorf("failed to scan source: %w", err)
		}

		data.Sources = append(data.Sources, source)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating source rows: %w", err)
	}

	// Get all anime sources
	rows, err = db.conn.Query(`
		SELECT 
			id, anime_id, source_id, source_anime_id
		FROM anime_source
	`)
	if err != nil {
		return fmt.Errorf("failed to query anime sources: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var animeSource AnimeSource
		err := rows.Scan(
			&animeSource.ID, &animeSource.AnimeID, &animeSource.SourceID,
			&animeSource.SourceAnimeID,
		)
		if err != nil {
			return fmt.Errorf("failed to scan anime source: %w", err)
		}

		data.AnimeSources = append(data.AnimeSources, animeSource)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating anime source rows: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for export: %w", err)
	}

	// Write to file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create export file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode export data: %w", err)
	}

	return nil
}

// ImportFromJSON imports data from a JSON file into the database
func (db *DB) ImportFromJSON(filePath string) error {
	// Read the file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open import file: %w", err)
	}
	defer file.Close()

	// Decode the JSON
	var data BackupData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return fmt.Errorf("failed to decode import data: %w", err)
	}

	// Start a transaction
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Import config entries
	for _, cfg := range data.Config {
		_, err := tx.Exec(
			`INSERT INTO config (key, value, updated_at) VALUES (?, ?, ?)
			ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = ?`,
			cfg.Key, cfg.Value, cfg.UpdatedAt,
			cfg.Value, cfg.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to import config entry %s: %w", cfg.Key, err)
		}
	}

	// Import anime
	for _, anime := range data.Anime {
		alternativeTitles, err := json.Marshal(anime.AlternativeTitles)
		if err != nil {
			return fmt.Errorf("failed to marshal alternative titles: %w", err)
		}

		genres, err := json.Marshal(anime.Genres)
		if err != nil {
			return fmt.Errorf("failed to marshal genres: %w", err)
		}

		// Import with original ID
		_, err = tx.Exec(
			`INSERT OR REPLACE INTO anime (
				id, title, original_title, alternative_titles, description, 
				total_episodes, type, year, season, status, genres, thumbnail_url,
				created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			anime.ID, anime.Title, anime.OriginalTitle, alternativeTitles, anime.Description,
			anime.TotalEpisodes, anime.Type, anime.Year, anime.Season, anime.Status,
			genres, anime.ThumbnailURL, anime.CreatedAt, anime.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to import anime %s: %w", anime.Title, err)
		}
	}

	// Import anime tracking
	for _, tracking := range data.AnimeTracking {
		_, err := tx.Exec(
			`INSERT OR REPLACE INTO anime_tracking (
				id, anime_id, tracker, tracker_id, status, score, 
				current_episode, total_episodes, last_updated
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			tracking.ID, tracking.AnimeID, tracking.Tracker, tracking.TrackerID, tracking.Status,
			tracking.Score, tracking.CurrentEpisode, tracking.TotalEpisodes, tracking.LastUpdated,
		)
		if err != nil {
			return fmt.Errorf("failed to import anime tracking for anime %d: %w", tracking.AnimeID, err)
		}
	}

	// Import episode progress
	for _, progress := range data.EpisodeProgress {
		_, err := tx.Exec(
			`INSERT OR REPLACE INTO episode_progress (
				id, anime_id, episode_number, position, duration, 
				playback_speed, watched, source_id, last_watched
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			progress.ID, progress.AnimeID, progress.EpisodeNumber, progress.Position, progress.Duration,
			progress.PlaybackSpeed, progress.Watched, progress.SourceID, progress.LastWatched,
		)
		if err != nil {
			return fmt.Errorf("failed to import episode progress for anime %d episode %f: %w",
				progress.AnimeID, progress.EpisodeNumber, err)
		}
	}

	// Import episodes
	for _, episode := range data.Episodes {
		_, err := tx.Exec(
			`INSERT OR REPLACE INTO episode (
				id, anime_id, number, title, description, duration, 
				thumbnail_url, air_date, is_filler, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			episode.ID, episode.AnimeID, episode.Number, episode.Title, episode.Description,
			episode.Duration, episode.ThumbnailURL, episode.AirDate, episode.IsFiller, episode.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to import episode %f for anime %d: %w",
				episode.Number, episode.AnimeID, err)
		}
	}

	// Import extensions
	for _, ext := range data.Extensions {
		_, err := tx.Exec(
			`INSERT OR REPLACE INTO extension (
				id, name, package, language, version, nsfw, path, repository_url,
				installed_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			ext.ID, ext.Name, ext.Package, ext.Language, ext.Version, ext.NSFW, ext.Path,
			ext.RepositoryURL, ext.InstalledAt, ext.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to import extension %s: %w", ext.Name, err)
		}
	}

	// Import sources
	for _, source := range data.Sources {
		_, err := tx.Exec(
			`INSERT OR REPLACE INTO source (
				id, source_id, extension_id, name, language, base_url, nsfw
			) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			source.ID, source.SourceID, source.ExtensionID, source.Name, source.Language,
			source.BaseURL, source.NSFW,
		)
		if err != nil {
			return fmt.Errorf("failed to import source %s: %w", source.Name, err)
		}
	}

	// Import anime sources
	for _, animeSource := range data.AnimeSources {
		_, err := tx.Exec(
			`INSERT OR REPLACE INTO anime_source (
				id, anime_id, source_id, source_anime_id
			) VALUES (?, ?, ?, ?)`,
			animeSource.ID, animeSource.AnimeID, animeSource.SourceID, animeSource.SourceAnimeID,
		)
		if err != nil {
			return fmt.Errorf("failed to import anime source mapping: %w", err)
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit import transaction: %w", err)
	}

	return nil
}
