package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Errors
var (
	ErrAnimeNotFound    = fmt.Errorf("anime not found")
	ErrTrackingNotFound = fmt.Errorf("tracking not found")
)

// Anime represents an anime in the database
type Anime struct {
	ID                int64
	Title             string
	OriginalTitle     string
	AlternativeTitles []string
	Description       string
	TotalEpisodes     int
	Type              string
	Year              int
	Season            string
	Status            string
	Genres            []string
	ThumbnailURL      string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// AnimeTracking represents a user's anime tracking information
type AnimeTracking struct {
	ID             int64
	AnimeID        int64
	Tracker        string
	TrackerID      string
	Status         string
	Score          float64
	CurrentEpisode float64
	TotalEpisodes  int
	LastUpdated    time.Time
}

// EpisodeProgress represents a user's episode viewing progress
type EpisodeProgress struct {
	ID            int64
	AnimeID       int64
	EpisodeNumber float64
	Position      int
	Duration      int
	PlaybackSpeed float64
	Watched       bool
	SourceID      string
	LastWatched   time.Time
}

// Episode represents an episode in the database
type Episode struct {
	ID           int64
	AnimeID      int64
	Number       float64
	Title        string
	Description  string
	Duration     int
	ThumbnailURL string
	AirDate      string
	IsFiller     bool
	CreatedAt    time.Time
}

// AddAnime adds a new anime to the database
func (db *DB) AddAnime(anime *Anime) error {
	// Convert slices to JSON
	alternativeTitles, err := json.Marshal(anime.AlternativeTitles)
	if err != nil {
		return err
	}

	genres, err := json.Marshal(anime.Genres)
	if err != nil {
		return err
	}

	// Insert anime
	result, err := db.conn.Exec(
		`INSERT INTO anime (
			title, original_title, alternative_titles, description, 
			total_episodes, type, year, season, status, genres, thumbnail_url
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		anime.Title, anime.OriginalTitle, alternativeTitles, anime.Description,
		anime.TotalEpisodes, anime.Type, anime.Year, anime.Season, anime.Status,
		genres, anime.ThumbnailURL,
	)
	if err != nil {
		return err
	}

	// Get the inserted ID
	anime.ID, err = result.LastInsertId()
	return err
}

// GetAnimeByID retrieves an anime by ID
func (db *DB) GetAnimeByID(id int64) (*Anime, error) {
	var anime Anime
	var alternativeTitlesJSON, genresJSON []byte

	err := db.conn.QueryRow(
		`SELECT 
			id, title, original_title, alternative_titles, description, 
			total_episodes, type, year, season, status, genres, thumbnail_url,
			created_at, updated_at
		FROM anime WHERE id = ?`, id,
	).Scan(
		&anime.ID, &anime.Title, &anime.OriginalTitle, &alternativeTitlesJSON,
		&anime.Description, &anime.TotalEpisodes, &anime.Type, &anime.Year,
		&anime.Season, &anime.Status, &genresJSON, &anime.ThumbnailURL,
		&anime.CreatedAt, &anime.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Parse JSON fields
	if err := json.Unmarshal(alternativeTitlesJSON, &anime.AlternativeTitles); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(genresJSON, &anime.Genres); err != nil {
		return nil, err
	}

	return &anime, nil
}

// GetAnimeByTitle retrieves an anime by title
func (db *DB) GetAnimeByTitle(title string) (*Anime, error) {
	var anime Anime
	var alternativeTitlesJSON, genresJSON []byte

	err := db.conn.QueryRow(
		`SELECT 
			id, title, original_title, alternative_titles, description, 
			total_episodes, type, year, season, status, genres, thumbnail_url,
			created_at, updated_at
		FROM anime WHERE title = ?`, title,
	).Scan(
		&anime.ID, &anime.Title, &anime.OriginalTitle, &alternativeTitlesJSON,
		&anime.Description, &anime.TotalEpisodes, &anime.Type, &anime.Year,
		&anime.Season, &anime.Status, &genresJSON, &anime.ThumbnailURL,
		&anime.CreatedAt, &anime.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Parse JSON fields
	if err := json.Unmarshal(alternativeTitlesJSON, &anime.AlternativeTitles); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(genresJSON, &anime.Genres); err != nil {
		return nil, err
	}

	return &anime, nil
}

// SearchAnime searches for anime by title
func (db *DB) SearchAnime(query string) ([]*Anime, error) {
	rows, err := db.conn.Query(
		`SELECT 
			id, title, original_title, alternative_titles, description, 
			total_episodes, type, year, season, status, genres, thumbnail_url,
			created_at, updated_at
		FROM anime 
		WHERE title LIKE ? OR original_title LIKE ?
		ORDER BY title`,
		"%"+query+"%", "%"+query+"%",
	)
	if err != nil {
		return nil, err
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
			return nil, err
		}

		// Parse JSON fields
		if err := json.Unmarshal(alternativeTitlesJSON, &anime.AlternativeTitles); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(genresJSON, &anime.Genres); err != nil {
			return nil, err
		}

		animes = append(animes, &anime)
	}

	return animes, rows.Err()
}

// UpdateAnime updates an existing anime
func (db *DB) UpdateAnime(anime *Anime) error {
	// Convert slices to JSON
	alternativeTitles, err := json.Marshal(anime.AlternativeTitles)
	if err != nil {
		return err
	}

	genres, err := json.Marshal(anime.Genres)
	if err != nil {
		return err
	}

	// Update anime
	_, err = db.conn.Exec(
		`UPDATE anime SET
			title = ?, original_title = ?, alternative_titles = ?, description = ?, 
			total_episodes = ?, type = ?, year = ?, season = ?, status = ?, 
			genres = ?, thumbnail_url = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		anime.Title, anime.OriginalTitle, alternativeTitles, anime.Description,
		anime.TotalEpisodes, anime.Type, anime.Year, anime.Season, anime.Status,
		genres, anime.ThumbnailURL, anime.ID,
	)
	return err
}

// DeleteAnime deletes an anime by ID
func (db *DB) DeleteAnime(id int64) error {
	_, err := db.conn.Exec("DELETE FROM anime WHERE id = ?", id)
	return err
}

// AddAnimeTracking adds or updates tracking information for an anime
func (db *DB) AddAnimeTracking(tracking *AnimeTracking) error {
	result, err := db.conn.Exec(
		`INSERT INTO anime_tracking (
			anime_id, tracker, tracker_id, status, score, 
			current_episode, total_episodes, last_updated
		) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(anime_id, tracker) DO UPDATE SET
			tracker_id = ?, status = ?, score = ?, 
			current_episode = ?, total_episodes = ?, last_updated = CURRENT_TIMESTAMP`,
		tracking.AnimeID, tracking.Tracker, tracking.TrackerID, tracking.Status,
		tracking.Score, tracking.CurrentEpisode, tracking.TotalEpisodes,
		tracking.TrackerID, tracking.Status, tracking.Score,
		tracking.CurrentEpisode, tracking.TotalEpisodes,
	)
	if err != nil {
		return err
	}

	// Get the inserted ID if this was a new tracking entry
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	if tracking.ID == 0 {
		tracking.ID = id
	}

	return nil
}

// GetAnimeTracking retrieves tracking information for an anime
func (db *DB) GetAnimeTracking(animeID int64, tracker string) (*AnimeTracking, error) {
	var tracking AnimeTracking

	err := db.conn.QueryRow(
		`SELECT 
			id, anime_id, tracker, tracker_id, status, score, 
			current_episode, total_episodes, last_updated
		FROM anime_tracking 
		WHERE anime_id = ? AND tracker = ?`,
		animeID, tracker,
	).Scan(
		&tracking.ID, &tracking.AnimeID, &tracking.Tracker, &tracking.TrackerID,
		&tracking.Status, &tracking.Score, &tracking.CurrentEpisode,
		&tracking.TotalEpisodes, &tracking.LastUpdated,
	)
	if err != nil {
		return nil, err
	}

	return &tracking, nil
}

// GetAllAnimeTracking retrieves all tracking information for an anime
func (db *DB) GetAllAnimeTracking(animeID int64) ([]*AnimeTracking, error) {
	rows, err := db.conn.Query(
		`SELECT 
			id, anime_id, tracker, tracker_id, status, score, 
			current_episode, total_episodes, last_updated
		FROM anime_tracking 
		WHERE anime_id = ?`,
		animeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trackings []*AnimeTracking
	for rows.Next() {
		var tracking AnimeTracking
		err := rows.Scan(
			&tracking.ID, &tracking.AnimeID, &tracking.Tracker, &tracking.TrackerID,
			&tracking.Status, &tracking.Score, &tracking.CurrentEpisode,
			&tracking.TotalEpisodes, &tracking.LastUpdated,
		)
		if err != nil {
			return nil, err
		}
		trackings = append(trackings, &tracking)
	}

	return trackings, rows.Err()
}

// DeleteAnimeTracking deletes tracking information for an anime
func (db *DB) DeleteAnimeTracking(animeID int64, tracker string) error {
	_, err := db.conn.Exec(
		"DELETE FROM anime_tracking WHERE anime_id = ? AND tracker = ?",
		animeID, tracker,
	)
	return err
}

// GetEpisodeProgress retrieves progress information for an episode
func (db *DB) GetEpisodeProgress(animeID int64, episodeNumber float64) (*EpisodeProgress, error) {
	var progress EpisodeProgress

	err := db.conn.QueryRow(
		`SELECT 
			id, anime_id, episode_number, position, duration, 
			playback_speed, watched, source_id, last_watched
		FROM episode_progress 
		WHERE anime_id = ? AND episode_number = ?`,
		animeID, episodeNumber,
	).Scan(
		&progress.ID, &progress.AnimeID, &progress.EpisodeNumber, &progress.Position,
		&progress.Duration, &progress.PlaybackSpeed, &progress.Watched,
		&progress.SourceID, &progress.LastWatched,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No progress yet
		}
		return nil, err
	}

	return &progress, nil
}

// GetAllEpisodeProgress retrieves progress information for all episodes of an anime
func (db *DB) GetAllEpisodeProgress(animeID int64) ([]*EpisodeProgress, error) {
	rows, err := db.conn.Query(
		`SELECT 
			id, anime_id, episode_number, position, duration, 
			playback_speed, watched, source_id, last_watched
		FROM episode_progress 
		WHERE anime_id = ?
		ORDER BY episode_number`,
		animeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var progressList []*EpisodeProgress
	for rows.Next() {
		var progress EpisodeProgress
		err := rows.Scan(
			&progress.ID, &progress.AnimeID, &progress.EpisodeNumber, &progress.Position,
			&progress.Duration, &progress.PlaybackSpeed, &progress.Watched,
			&progress.SourceID, &progress.LastWatched,
		)
		if err != nil {
			return nil, err
		}
		progressList = append(progressList, &progress)
	}

	return progressList, rows.Err()
}

// DeleteEpisodeProgress deletes progress information for an episode
func (db *DB) DeleteEpisodeProgress(animeID int64, episodeNumber float64) error {
	_, err := db.conn.Exec(
		"DELETE FROM episode_progress WHERE anime_id = ? AND episode_number = ?",
		animeID, episodeNumber,
	)
	return err
}

// GetLastWatchedAnime retrieves the last watched anime
func (db *DB) GetLastWatchedAnime() (*Anime, error) {
	var animeID int64

	err := db.conn.QueryRow(`
		SELECT anime_id FROM episode_progress
		ORDER BY last_watched DESC
		LIMIT 1
	`).Scan(&animeID)
	if err != nil {
		return nil, err
	}

	return db.GetAnimeByID(animeID)
}

// GetWatchingAnime retrieves all anime that the user is currently watching
func (db *DB) GetWatchingAnime() ([]*Anime, error) {
	rows, err := db.conn.Query(`
		SELECT DISTINCT a.* FROM anime a
		JOIN anime_tracking t ON a.id = t.anime_id
		WHERE t.status = 'watching'
		ORDER BY t.last_updated DESC
	`)
	if err != nil {
		return nil, err
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
			return nil, err
		}

		// Parse JSON fields
		if err := json.Unmarshal(alternativeTitlesJSON, &anime.AlternativeTitles); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(genresJSON, &anime.Genres); err != nil {
			return nil, err
		}

		animes = append(animes, &anime)
	}

	return animes, rows.Err()
}

// GetEpisode retrieves an episode by anime ID and episode number
func (db *DB) GetEpisode(animeID int64, number float64) (*Episode, error) {
	var episode Episode

	err := db.conn.QueryRow(
		`SELECT 
			id, anime_id, number, title, description, duration, 
			thumbnail_url, air_date, is_filler, created_at
		FROM episode 
		WHERE anime_id = ? AND number = ?`,
		animeID, number,
	).Scan(
		&episode.ID, &episode.AnimeID, &episode.Number, &episode.Title,
		&episode.Description, &episode.Duration, &episode.ThumbnailURL,
		&episode.AirDate, &episode.IsFiller, &episode.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &episode, nil
}

// GetAllEpisodes retrieves all episodes for an anime
func (db *DB) GetAllEpisodes(animeID int64) ([]*Episode, error) {
	rows, err := db.conn.Query(
		`SELECT 
			id, anime_id, number, title, description, duration, 
			thumbnail_url, air_date, is_filler, created_at
		FROM episode 
		WHERE anime_id = ?
		ORDER BY number`,
		animeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []*Episode
	for rows.Next() {
		var episode Episode
		err := rows.Scan(
			&episode.ID, &episode.AnimeID, &episode.Number, &episode.Title,
			&episode.Description, &episode.Duration, &episode.ThumbnailURL,
			&episode.AirDate, &episode.IsFiller, &episode.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		episodes = append(episodes, &episode)
	}

	return episodes, rows.Err()
}

// DeleteEpisode deletes an episode
func (db *DB) DeleteEpisode(animeID int64, number float64) error {
	_, err := db.conn.Exec(
		"DELETE FROM episode WHERE anime_id = ? AND number = ?",
		animeID, number,
	)
	return err
}

// GetAnimeByExternalID gets an anime by its external ID (e.g., MAL, Anilist)
func (db *DB) GetAnimeByExternalID(externalID string, tracker string) (*Anime, error) {
	// First, check if we have a tracking entry for this external ID
	query := `
		SELECT anime_id FROM anime_tracking
		WHERE tracker = ? AND tracker_id = ?
	`

	var animeID int64
	err := db.conn.QueryRow(query, tracker, externalID).Scan(&animeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrAnimeNotFound
		}
		return nil, fmt.Errorf("failed to query anime tracking: %w", err)
	}

	// Now get the anime by ID
	return db.GetAnime(animeID)
}

// GetAllAnimeTrackingByTracker gets all anime tracking entries for a specific tracker
func (db *DB) GetAllAnimeTrackingByTracker(tracker string) ([]*AnimeTracking, error) {
	query := `
		SELECT id, anime_id, tracker, tracker_id, status, score, current_episode, total_episodes, last_updated
		FROM anime_tracking
		WHERE tracker = ?
	`

	rows, err := db.conn.Query(query, tracker)
	if err != nil {
		return nil, fmt.Errorf("failed to query anime tracking: %w", err)
	}
	defer rows.Close()

	var trackings []*AnimeTracking
	for rows.Next() {
		tracking := &AnimeTracking{}
		err := rows.Scan(
			&tracking.ID,
			&tracking.AnimeID,
			&tracking.Tracker,
			&tracking.TrackerID,
			&tracking.Status,
			&tracking.Score,
			&tracking.CurrentEpisode,
			&tracking.TotalEpisodes,
			&tracking.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan anime tracking: %w", err)
		}
		trackings = append(trackings, tracking)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating anime tracking rows: %w", err)
	}

	return trackings, nil
}

// GetAllAnimeTrackingByAnimeID gets all anime tracking entries for a specific anime
func (db *DB) GetAllAnimeTrackingByAnimeID(animeID int64) ([]*AnimeTracking, error) {
	query := `
		SELECT id, anime_id, tracker, tracker_id, status, score, current_episode, total_episodes, last_updated
		FROM anime_tracking
		WHERE anime_id = ?
	`

	rows, err := db.conn.Query(query, animeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query anime tracking: %w", err)
	}
	defer rows.Close()

	var trackings []*AnimeTracking
	for rows.Next() {
		tracking := &AnimeTracking{}
		err := rows.Scan(
			&tracking.ID,
			&tracking.AnimeID,
			&tracking.Tracker,
			&tracking.TrackerID,
			&tracking.Status,
			&tracking.Score,
			&tracking.CurrentEpisode,
			&tracking.TotalEpisodes,
			&tracking.LastUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan anime tracking: %w", err)
		}
		trackings = append(trackings, tracking)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating anime tracking rows: %w", err)
	}

	return trackings, nil
}

// GetAnime gets an anime by ID
func (db *DB) GetAnime(id int64) (*Anime, error) {
	query := `
		SELECT id, title, original_title, alternative_titles, description, total_episodes,
		       type, year, season, status, genres, thumbnail_url, created_at, updated_at
		FROM anime
		WHERE id = ?
	`

	var (
		anime                 = &Anime{}
		alternativeTitlesJSON string
		genresJSON            string
	)

	err := db.conn.QueryRow(query, id).Scan(
		&anime.ID,
		&anime.Title,
		&anime.OriginalTitle,
		&alternativeTitlesJSON,
		&anime.Description,
		&anime.TotalEpisodes,
		&anime.Type,
		&anime.Year,
		&anime.Season,
		&anime.Status,
		&genresJSON,
		&anime.ThumbnailURL,
		&anime.CreatedAt,
		&anime.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrAnimeNotFound
		}
		return nil, fmt.Errorf("failed to query anime: %w", err)
	}

	// Parse JSON fields
	if alternativeTitlesJSON != "" {
		if err := json.Unmarshal([]byte(alternativeTitlesJSON), &anime.AlternativeTitles); err != nil {
			return nil, fmt.Errorf("failed to parse alternative titles: %w", err)
		}
	}

	if genresJSON != "" {
		if err := json.Unmarshal([]byte(genresJSON), &anime.Genres); err != nil {
			return nil, fmt.Errorf("failed to parse genres: %w", err)
		}
	}

	return anime, nil
}

// GetAllAnime retrieves all anime from the database
func (db *DB) GetAllAnime() ([]*Anime, error) {
	rows, err := db.conn.Query(`
		SELECT id, title, original_title, alternative_titles, description, 
		       total_episodes, type, year, season, status, genres, thumbnail_url,
		       created_at, updated_at
		FROM anime 
		ORDER BY title
	`)
	if err != nil {
		return nil, err
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
			return nil, err
		}

		// Parse JSON fields
		if err := json.Unmarshal(alternativeTitlesJSON, &anime.AlternativeTitles); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(genresJSON, &anime.Genres); err != nil {
			return nil, err
		}

		animes = append(animes, &anime)
	}

	return animes, rows.Err()
}
