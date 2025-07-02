package appcore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/wraient/pair/pkg/config"
	"github.com/wraient/pair/pkg/database"
	"github.com/wraient/pair/pkg/tracker"
	"github.com/wraient/pair/pkg/ui"
)

// handleCurrentlyWatching handles the currently watching list view
func (a *App) handleCurrentlyWatching(ctx context.Context) error {
	// Get the database connection
	db := config.GetDB()

	var syncErrors []error

	// Sync with all available trackers (same as Show All)
	err := a.syncWithTrackers(ctx, db, &syncErrors)
	if err != nil {
		return fmt.Errorf("failed to sync with trackers: %w", err)
	}

	// Get currently watching anime from the database after sync
	entries, err := db.GetCurrentlyWatchingAnime()
	if err != nil {
		return fmt.Errorf("failed to get watching entries: %w", err)
	}

	// Convert to UserAnimeEntry format
	watchingEntries := make([]tracker.UserAnimeEntry, 0)

	for _, entry := range entries {
		// Try to get tracking info from the primary service first
		var tracking *database.AnimeTracking
		var err error

		if a.config.Tracking.Service != "" {
			tracking, err = db.GetAnimeTracking(entry.ID, string(a.config.Tracking.Service))
			if err != nil && err != sql.ErrNoRows {
				return fmt.Errorf("failed to get tracking info: %w", err)
			}
		}

		// If no tracking from primary service, get from any available tracker
		if tracking == nil {
			allTrackings, err := db.GetAllAnimeTracking(entry.ID)
			if err != nil {
				return fmt.Errorf("failed to get all tracking info: %w", err)
			}

			// Use the most recently updated tracking info
			for _, t := range allTrackings {
				if tracking == nil || t.LastUpdated.After(tracking.LastUpdated) {
					tracking = t
				}
			}
		}

		userEntry := tracker.UserAnimeEntry{
			AnimeInfo: tracker.AnimeInfo{
				ID:                strconv.FormatInt(entry.ID, 10),
				Title:             entry.Title,
				EnglishTitle:      entry.OriginalTitle,
				AlternativeTitles: entry.AlternativeTitles,
				Synopsis:          entry.Description,
				Type:              entry.Type,
				Episodes:          entry.TotalEpisodes,
				Status:            entry.Status,
				Year:              entry.Year,
				Season:            entry.Season,
				Genres:            entry.Genres,
				ImageURL:          entry.ThumbnailURL,
			},
			Status:   tracker.Status(entry.Status),
			Progress: 0,
		}

		if tracking != nil {
			userEntry.Progress = tracking.CurrentEpisode
			userEntry.Score = tracking.Score
			userEntry.LastUpdated = tracking.LastUpdated
		}

		watchingEntries = append(watchingEntries, userEntry)
	}

	// If we have any entries, show them
	if len(watchingEntries) > 0 {
		selectedID, err := ui.ShowAnimeList(watchingEntries)
		if err != nil {
			return fmt.Errorf("failed to show anime list: %w", err)
		}

		// Get selected anime details
		var selectedAnime *tracker.AnimeInfo
		for _, entry := range watchingEntries {
			if entry.ID == selectedID {
				selectedAnime = &entry.AnimeInfo
				break
			}
		}

		if selectedAnime == nil {
			return fmt.Errorf("selected anime not found")
		}

		// Show anime update menu
		action, err := ui.ShowAnimeUpdateMenu(selectedAnime)
		if err != nil {
			return fmt.Errorf("failed to show update menu: %w", err)
		}

		// Get the active tracker
		t, err := a.trackerMgr.GetTracker(string(a.config.Tracking.Service))
		if err != nil {
			return fmt.Errorf("failed to get tracker: %w", err)
		}

		// Handle selected action
		switch action {
		case "status":
			status, err := ui.ShowAnimeStatusSelection()
			if err != nil {
				return err
			}
			return t.UpdateAnimeStatus(ctx, selectedID, status, 0, 0)
		case "progress":
			episode, err := ui.ShowEpisodeSelection(selectedAnime.Episodes)
			if err != nil {
				return err
			}
			return t.UpdateAnimeStatus(ctx, selectedID, "", episode, 0)
		case "score":
			score, err := ui.ShowAnimeScoreSelection()
			if err != nil {
				return err
			}
			return t.UpdateAnimeStatus(ctx, selectedID, "", 0, score)
		}
	} else {
		fmt.Println("No currently watching anime found")
	}

	// Log any sync errors that occurred
	if len(syncErrors) > 0 {
		fmt.Println("\nSync errors occurred:")
		for _, err := range syncErrors {
			fmt.Printf("- %v\n", err)
		}
	}

	return nil
}
