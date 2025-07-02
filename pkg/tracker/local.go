package tracker

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/wraient/pair/pkg/database"
)

// LocalTracker implements the Tracker interface for local tracking
type LocalTracker struct {
	db *database.DB
}

// NewLocalTracker creates a new LocalTracker
func NewLocalTracker(db *database.DB) *LocalTracker {
	return &LocalTracker{
		db: db,
	}
}

// Name returns the name of the tracker
func (t *LocalTracker) Name() string {
	return "local"
}

// IsAuthenticated checks if the user is authenticated with the tracker
// Local tracker is always authenticated
func (t *LocalTracker) IsAuthenticated() bool {
	return true
}

// Authenticate authenticates the user with the tracker
// Local tracker doesn't need authentication
func (t *LocalTracker) Authenticate(ctx context.Context) error {
	return nil
}

// SearchAnime searches for anime locally
func (t *LocalTracker) SearchAnime(ctx context.Context, query string, limit int) ([]AnimeInfo, error) {
	// Search anime in the database
	animes, err := t.db.SearchAnime(query)
	if err != nil {
		return nil, fmt.Errorf("failed to search anime: %w", err)
	}

	// Convert database.Anime to tracker.AnimeInfo
	results := make([]AnimeInfo, 0, len(animes))
	for _, anime := range animes {
		info := AnimeInfo{
			ID:                fmt.Sprintf("%d", anime.ID),
			Title:             anime.Title,
			EnglishTitle:      anime.Title, // Use title as English title by default
			JapaneseTitle:     anime.OriginalTitle,
			AlternativeTitles: anime.AlternativeTitles,
			Synopsis:          anime.Description,
			Type:              anime.Type,
			Status:            anime.Status,
			Episodes:          anime.TotalEpisodes,
			Year:              anime.Year,
			Season:            anime.Season,
			Genres:            anime.Genres,
			ImageURL:          anime.ThumbnailURL,
		}
		results = append(results, info)
	}

	return results, nil
}

// GetAnimeDetails gets detailed information about an anime
func (t *LocalTracker) GetAnimeDetails(ctx context.Context, id string) (*AnimeInfo, error) {
	// Get anime by ID
	animeID := 0
	fmt.Sscanf(id, "%d", &animeID)

	anime, err := t.db.GetAnime(int64(animeID))
	if err != nil {
		return nil, fmt.Errorf("failed to get anime details: %w", err)
	}

	// Convert database.Anime to tracker.AnimeInfo
	info := &AnimeInfo{
		ID:                fmt.Sprintf("%d", anime.ID),
		Title:             anime.Title,
		EnglishTitle:      anime.Title, // Use title as English title by default
		JapaneseTitle:     anime.OriginalTitle,
		AlternativeTitles: anime.AlternativeTitles,
		Synopsis:          anime.Description,
		Type:              anime.Type,
		Status:            anime.Status,
		Episodes:          anime.TotalEpisodes,
		Year:              anime.Year,
		Season:            anime.Season,
		Genres:            anime.Genres,
		ImageURL:          anime.ThumbnailURL,
	}

	return info, nil
}

// GetUserAnimeList gets the user's anime list
func (t *LocalTracker) GetUserAnimeList(ctx context.Context) ([]UserAnimeEntry, error) {
	// Get all anime tracking entries for local tracker
	trackings, err := t.db.GetAllAnimeTrackingByTracker(t.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to get user anime list: %w", err)
	}

	entries := make([]UserAnimeEntry, 0, len(trackings))
	for _, tracking := range trackings {
		// Get anime details
		anime, err := t.db.GetAnime(tracking.AnimeID)
		if err != nil {
			continue
		}

		// Map status string to enum
		status := StatusWatching
		switch tracking.Status {
		case "watching":
			status = StatusWatching
		case "completed":
			status = StatusCompleted
		case "on_hold":
			status = StatusOnHold
		case "dropped":
			status = StatusDropped
		case "plan_to_watch":
			status = StatusPlanToWatch
		}

		entry := UserAnimeEntry{
			AnimeInfo: AnimeInfo{
				ID:                fmt.Sprintf("%d", anime.ID),
				Title:             anime.Title,
				EnglishTitle:      anime.Title,
				JapaneseTitle:     anime.OriginalTitle,
				AlternativeTitles: anime.AlternativeTitles,
				Synopsis:          anime.Description,
				Type:              anime.Type,
				Status:            anime.Status,
				Episodes:          anime.TotalEpisodes,
				Year:              anime.Year,
				Season:            anime.Season,
				Genres:            anime.Genres,
				ImageURL:          anime.ThumbnailURL,
			},
			Status:      status,
			Score:       tracking.Score,
			Progress:    tracking.CurrentEpisode,
			LastUpdated: tracking.LastUpdated,
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// UpdateAnimeStatus updates the watch status of an anime
func (t *LocalTracker) UpdateAnimeStatus(ctx context.Context, id string, status Status, episode float64, score float64) error {
	// Get anime by ID
	animeID := 0
	fmt.Sscanf(id, "%d", &animeID)

	// Get tracking entry
	tracking, err := t.db.GetAnimeTracking(int64(animeID), t.Name())
	if err != nil {
		if err == sql.ErrNoRows {
			// Create new tracking entry
			tracking = &database.AnimeTracking{
				AnimeID:        int64(animeID),
				Tracker:        t.Name(),
				TrackerID:      id,
				Status:         string(status),
				Score:          score,
				CurrentEpisode: episode,
				LastUpdated:    time.Now(),
			}

			err := t.db.AddAnimeTracking(tracking)
			if err != nil {
				return fmt.Errorf("failed to create tracking entry: %w", err)
			}

			return nil
		}

		return fmt.Errorf("failed to get tracking entry: %w", err)
	}

	// Update tracking entry
	tracking.Status = string(status)
	if episode > 0 {
		tracking.CurrentEpisode = episode
	}
	if score > 0 {
		tracking.Score = score
	}
	tracking.LastUpdated = time.Now()

	if err := t.db.UpdateAnimeTrackingObject(tracking); err != nil {
		return fmt.Errorf("failed to update tracking entry: %w", err)
	}

	return nil
}

// SyncFromRemote synchronizes the local database with the remote tracker
// Local tracker doesn't need to sync from remote
func (t *LocalTracker) SyncFromRemote(ctx context.Context, db *database.DB) (SyncStats, error) {
	return SyncStats{}, nil
}

// SyncToRemote synchronizes the remote tracker with the local database
// Local tracker doesn't need to sync to remote
func (t *LocalTracker) SyncToRemote(ctx context.Context, db *database.DB) (SyncStats, error) {
	return SyncStats{}, nil
}
