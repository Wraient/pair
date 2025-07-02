package database

import (
	"time"
)

// GetAllAnimeSources retrieves all anime sources
func (db *DB) GetAllAnimeSources(sourceID int64) ([]*AnimeSource, error) {
	rows, err := db.conn.Query(
		`SELECT 
			id, anime_id, source_id, source_anime_id
		FROM anime_source WHERE source_id = ?`,
		sourceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []*AnimeSource
	for rows.Next() {
		var source AnimeSource
		err := rows.Scan(
			&source.ID, &source.AnimeID, &source.SourceID, &source.SourceAnimeID,
		)
		if err != nil {
			return nil, err
		}

		sources = append(sources, &source)
	}

	return sources, rows.Err()
}

// AddEpisode adds a new episode to the database
func (db *DB) AddEpisode(episode *Episode) error {
	result, err := db.conn.Exec(
		`INSERT INTO episode (
			anime_id, number, title, thumbnail_url
		) VALUES (?, ?, ?, ?)
		ON CONFLICT(anime_id, number) DO UPDATE SET
			title = ?, thumbnail_url = ?`,
		episode.AnimeID, episode.Number, episode.Title, episode.ThumbnailURL,
		episode.Title, episode.ThumbnailURL,
	)
	if err != nil {
		return err
	}

	// Get the inserted ID if this was a new episode
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	if episode.ID == 0 {
		episode.ID = id
	}

	return nil
}

// AddEpisodeProgress adds or updates episode progress
func (db *DB) AddEpisodeProgress(progress *EpisodeProgress) error {
	result, err := db.conn.Exec(
		`INSERT INTO episode_progress (
			anime_id, episode_number, position, duration, 
			playback_speed, watched, source_id, last_watched
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(anime_id, episode_number) DO UPDATE SET
			position = ?, duration = ?, playback_speed = ?, 
			watched = ?, source_id = ?, last_watched = ?`,
		progress.AnimeID, progress.EpisodeNumber, progress.Position, progress.Duration,
		progress.PlaybackSpeed, progress.Watched, progress.SourceID, progress.LastWatched,
		progress.Position, progress.Duration, progress.PlaybackSpeed,
		progress.Watched, progress.SourceID, progress.LastWatched,
	)
	if err != nil {
		return err
	}

	// Get the inserted ID if this was a new progress entry
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	if progress.ID == 0 {
		progress.ID = id
	}

	return nil
}

// UpdateAnimeTracking updates the current episode for anime tracking
func (db *DB) UpdateAnimeTracking(animeID int64, tracker string, currentEpisode float64) error {
	_, err := db.conn.Exec(
		`UPDATE anime_tracking 
		SET current_episode = ?, last_updated = CURRENT_TIMESTAMP
		WHERE anime_id = ? AND tracker = ? AND current_episode < ?`,
		currentEpisode, animeID, tracker, currentEpisode,
	)
	return err
}

// UpdateAnimeTrackingObject updates a full anime tracking object
func (db *DB) UpdateAnimeTrackingObject(tracking *AnimeTracking) error {
	_, err := db.conn.Exec(
		`UPDATE anime_tracking
		SET tracker_id = ?, status = ?, score = ?, 
		    current_episode = ?, total_episodes = ?, last_updated = ?
		WHERE id = ?`,
		tracking.TrackerID, tracking.Status, tracking.Score,
		tracking.CurrentEpisode, tracking.TotalEpisodes, time.Now(),
		tracking.ID,
	)
	return err
}
