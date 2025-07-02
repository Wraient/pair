package appcore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/wraient/pair/pkg/config"
	"github.com/wraient/pair/pkg/tracker"
	"github.com/wraient/pair/pkg/ui"
)

// handleCurrentlyWatching handles the currently watching list view
func (a *App) handleCurrentlyWatching(ctx context.Context) error {
	// Get the database connection
	db := config.GetDB()

	// Get all anime from the database first
	entries, err := db.GetCurrentlyWatchingAnime()
	if err != nil {
		return fmt.Errorf("failed to get watching entries: %w", err)
	}

	// Convert to UserAnimeEntry format and create a map for easy lookup
	watchingEntries := make([]tracker.UserAnimeEntry, 0)
	localEntriesMap := make(map[string]tracker.UserAnimeEntry)

	for _, entry := range entries {
		tracking, err := db.GetAnimeTracking(entry.ID, string(a.config.Tracking.Service))
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("failed to get tracking info: %w", err)
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
		localEntriesMap[userEntry.ID] = userEntry
	}

	// Try to sync with remote trackers if available
	t, err := a.trackerMgr.GetTracker(string(a.config.Tracking.Service))
	if err == nil && t.IsAuthenticated() {
		remoteEntries, err := t.GetUserAnimeList(ctx)
		if err == nil {
			for _, remoteEntry := range remoteEntries {
				if remoteEntry.Status != tracker.StatusWatching {
					continue
				}

				localEntry, exists := localEntriesMap[remoteEntry.ID]
				if !exists {
					// New entry from remote, add to local
					watchingEntries = append(watchingEntries, remoteEntry)
					// Update local database
					err := t.UpdateAnimeStatus(ctx, remoteEntry.ID, remoteEntry.Status, remoteEntry.Progress, remoteEntry.Score)
					if err != nil {
						fmt.Printf("Failed to update local database for new entry %s: %v\n", remoteEntry.Title, err)
					}
				} else {
					// Entry exists in both places, handle sync
					if remoteEntry.LastUpdated.After(localEntry.LastUpdated) {
						// Remote is newer
						if remoteEntry.Progress > localEntry.Progress {
							// Remote has higher episode count, update local
							err := t.UpdateAnimeStatus(ctx, remoteEntry.ID, remoteEntry.Status, remoteEntry.Progress, remoteEntry.Score)
							if err != nil {
								fmt.Printf("Failed to update local progress for %s: %v\n", remoteEntry.Title, err)
							}
							// Update the entry in our list
							for i, entry := range watchingEntries {
								if entry.ID == remoteEntry.ID {
									watchingEntries[i] = remoteEntry
									break
								}
							}
						}
					} else {
						// Local is newer
						if localEntry.Progress > remoteEntry.Progress {
							// Local has higher episode count, update remote
							err := t.UpdateAnimeStatus(ctx, localEntry.ID, localEntry.Status, localEntry.Progress, localEntry.Score)
							if err != nil {
								fmt.Printf("Failed to update remote progress for %s: %v\n", localEntry.Title, err)
							}
						}
					}
				}
			}
		}
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

	return nil
}
