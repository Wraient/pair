package appcore

import (
	"context"
	"fmt"
	"strings"

	"github.com/wraient/pair/pkg/config"
	"github.com/wraient/pair/pkg/database"
	"github.com/wraient/pair/pkg/tracker"
	"github.com/wraient/pair/pkg/ui"
)

// setupMainMenu creates and configures the main menu
func (a *App) setupMainMenu() *ui.Menu {
	mainMenu := ui.NewMenu("Main Menu", ui.List)

	// Continue watching
	mainMenu.AddItem("Continue watching", "continue", func(ctx context.Context) error {
		// TODO: Implement continue watching functionality
		return nil
	}).SetDescription("Continue watching your last anime")

	// Currently watching
	mainMenu.AddItem("Currently watching", "watching", func(ctx context.Context) error {
		return a.handleCurrentlyWatching(ctx)
	}).SetDescription("Show your currently watching anime")

	// Show all anime
	mainMenu.AddItem("Show all anime", "list", func(ctx context.Context) error {
		return a.handleAnimeList(ctx)
	}).SetDescription("Browse your complete anime list")

	// Settings submenu
	settingsMenu := a.setupSettingsMenu()
	mainMenu.AddItem("Settings", "settings", nil).
		AddSubMenu(settingsMenu).
		SetDescription("Configure application settings")

	// Extensions submenu
	extensionsMenu := a.setupExtensionsMenu()
	mainMenu.AddItem("Extensions", "extensions", nil).
		AddSubMenu(extensionsMenu).
		SetDescription("Manage extensions and sources")

	// Quit
	mainMenu.AddItem("Quit", "quit", func(ctx context.Context) error {
		return nil // Returning nil will exit the application
	}).SetDescription("Exit the application")

	return mainMenu
}

// setupSettingsMenu creates and configures the settings menu
func (a *App) setupSettingsMenu() *ui.Menu {
	settingsMenu := ui.NewMenu("Settings", ui.List)

	// Anilist settings
	settingsMenu.AddItem("Login Anilist", "login_anilist", func(ctx context.Context) error {
		t, err := a.trackerMgr.GetTracker("anilist")
		if err != nil {
			return fmt.Errorf("failed to get Anilist tracker: %w", err)
		}
		return t.Authenticate(ctx)
	}).SetDescription("Configure Anilist integration")

	// Add more settings items here...

	return settingsMenu
}

// setupExtensionsMenu creates and configures the extensions menu
func (a *App) setupExtensionsMenu() *ui.Menu {
	extensionsMenu := ui.NewMenu("Extensions", ui.List)

	// Add extension-related menu items here...

	return extensionsMenu
}

// handleAnimeList handles the anime list view with comprehensive sync
func (a *App) handleAnimeList(ctx context.Context) error {
	// Get the database connection
	db := config.GetDB()

	var syncErrors []error

	// Sync with all available trackers
	err := a.syncWithTrackers(ctx, db, &syncErrors)
	if err != nil {
		return fmt.Errorf("failed to sync with trackers: %w", err)
	}

	// Get all anime from local database for display
	allAnime, err := db.GetAllAnime()
	if err != nil {
		return fmt.Errorf("failed to get anime from database: %w", err)
	}

	// Convert to display format
	var displayEntries []tracker.UserAnimeEntry
	for _, anime := range allAnime {
		// Get tracking info for the primary service
		var primaryTracking *database.AnimeTracking
		if a.config.Tracking.Service != "" {
			primaryTracking, _ = db.GetAnimeTracking(anime.ID, string(a.config.Tracking.Service))
		}

		// Create display entry
		userEntry := tracker.UserAnimeEntry{
			AnimeInfo: tracker.AnimeInfo{
				ID:                fmt.Sprintf("%d", anime.ID),
				Title:             anime.Title,
				EnglishTitle:      anime.OriginalTitle,
				AlternativeTitles: anime.AlternativeTitles,
				Synopsis:          anime.Description,
				Type:              anime.Type,
				Episodes:          anime.TotalEpisodes,
				Status:            anime.Status,
				Year:              anime.Year,
				Season:            anime.Season,
				Genres:            anime.Genres,
				ImageURL:          anime.ThumbnailURL,
			},
			Status: tracker.Status(anime.Status),
		}

		if primaryTracking != nil {
			userEntry.Progress = primaryTracking.CurrentEpisode
			userEntry.Score = primaryTracking.Score
			userEntry.LastUpdated = primaryTracking.LastUpdated
		}

		displayEntries = append(displayEntries, userEntry)
	}

	// Show anime list if we have entries
	if len(displayEntries) > 0 {
		// Create menu items
		menuItems := make([]ui.Pair, 0, len(displayEntries)+1)

		// Add all anime entries
		for _, entry := range displayEntries {
			// Create a display string with title and additional info
			displayInfo := []string{entry.Title}
			if entry.Episodes > 0 {
				displayInfo = append(displayInfo, fmt.Sprintf("%d/%d eps", int(entry.Progress), entry.Episodes))
			}
			if entry.Score > 0 {
				displayInfo = append(displayInfo, fmt.Sprintf("Score: %.1f", entry.Score))
			}
			if entry.Status != "" {
				displayInfo = append(displayInfo, string(entry.Status))
			}

			menuItems = append(menuItems, ui.Pair{
				Label: strings.Join(displayInfo, " - "),
				Value: entry.ID,
			})
		}

		// Add back option
		menuItems = append(menuItems, ui.Pair{
			Label: "Back",
			Value: "back",
		})

		// Show menu
		selectedID, err := ui.OpenMenu(ui.List, menuItems)
		if err != nil {
			return fmt.Errorf("failed to show menu: %w", err)
		}

		// If back was selected, return
		if selectedID == "back" {
			return nil
		}

		// Get selected anime details
		var selectedAnime *tracker.AnimeInfo
		for _, entry := range displayEntries {
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
		if a.config.Tracking.Service != "" {
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
			case "back":
				return nil
			}
		}
	} else {
		fmt.Println("No anime found in database")
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

// syncWithTrackers syncs anime data with all authenticated trackers
func (a *App) syncWithTrackers(ctx context.Context, db *database.DB, syncErrors *[]error) error {
	// Get all available trackers
	availableTrackers := []string{"anilist", "mal"}

	for _, trackerName := range availableTrackers {
		animeTracker, err := a.trackerMgr.GetTracker(trackerName)
		if err != nil {
			continue // Skip if tracker not available
		}

		// Check if tracker is authenticated
		if !animeTracker.IsAuthenticated() {
			continue // Skip if not authenticated
		}

		err = a.syncWithSingleTracker(ctx, db, animeTracker, trackerName, syncErrors)
		if err != nil {
			*syncErrors = append(*syncErrors, fmt.Errorf("failed to sync with %s: %w", trackerName, err))
		}
	}

	return nil
}

// syncWithSingleTracker syncs anime data with a single tracker
func (a *App) syncWithSingleTracker(ctx context.Context, db *database.DB, animeTracker tracker.Tracker, trackerName string, syncErrors *[]error) error {
	// Get remote entries from tracker
	remoteEntries, err := animeTracker.GetUserAnimeList(ctx)
	if err != nil {
		return fmt.Errorf("failed to get anime list: %w", err)
	}

	// Create map of remote entries for easy lookup
	remoteEntriesMap := make(map[string]*tracker.UserAnimeEntry)
	for i := range remoteEntries {
		remoteEntriesMap[remoteEntries[i].ID] = &remoteEntries[i]
	}

	// Get all local tracking entries for this tracker
	localTrackings, err := db.GetAllAnimeTrackingByTracker(trackerName)
	if err != nil {
		return fmt.Errorf("failed to get local tracking: %w", err)
	}

	// Create map of local entries for easy lookup
	localTrackingMap := make(map[string]*database.AnimeTracking)
	for _, tracking := range localTrackings {
		localTrackingMap[tracking.TrackerID] = tracking
	}

	// Process each remote entry
	for _, entry := range remoteEntries {
		err := a.processRemoteEntry(ctx, db, &entry, trackerName, localTrackingMap, syncErrors)
		if err != nil {
			*syncErrors = append(*syncErrors, fmt.Errorf("failed to process remote entry %s: %w", entry.Title, err))
		}
	}

	// Check for local entries that are not in remote (deleted from remote)
	for trackerID, localTracking := range localTrackingMap {
		if _, exists := remoteEntriesMap[trackerID]; !exists {
			// Entry was deleted from remote, delete from local
			err := a.deleteLocalEntry(db, localTracking, trackerName)
			if err != nil {
				*syncErrors = append(*syncErrors, fmt.Errorf("failed to delete local entry for tracker ID %s: %w", trackerID, err))
			}
		}
	}

	return nil
}

// processRemoteEntry processes a single remote anime entry
func (a *App) processRemoteEntry(ctx context.Context, db *database.DB, entry *tracker.UserAnimeEntry, trackerName string, localTrackingMap map[string]*database.AnimeTracking, syncErrors *[]error) error {
	// Check if anime exists in database
	anime, err := db.GetAnimeByExternalID(entry.ID, trackerName)
	if err != nil && err != database.ErrAnimeNotFound {
		return fmt.Errorf("failed to check anime: %w", err)
	}

	if anime == nil {
		// Anime doesn't exist, add it
		animeData := &database.Anime{
			Title:             entry.Title,
			OriginalTitle:     entry.EnglishTitle,
			AlternativeTitles: entry.AlternativeTitles,
			Description:       entry.Synopsis,
			TotalEpisodes:     entry.Episodes,
			Type:              entry.Type,
			Year:              entry.Year,
			Season:            entry.Season,
			Status:            string(entry.Status),
			Genres:            entry.Genres,
			ThumbnailURL:      entry.ImageURL,
		}

		if err := db.AddAnime(animeData); err != nil {
			return fmt.Errorf("failed to add anime to local db: %w", err)
		}

		// Add tracking information
		tracking := &database.AnimeTracking{
			AnimeID:        animeData.ID,
			Tracker:        trackerName,
			TrackerID:      entry.ID,
			Status:         string(entry.Status),
			Score:          entry.Score,
			CurrentEpisode: entry.Progress,
			TotalEpisodes:  entry.Episodes,
			LastUpdated:    entry.LastUpdated,
		}

		if err := db.AddAnimeTracking(tracking); err != nil {
			return fmt.Errorf("failed to add tracking: %w", err)
		}
	} else {
		// Anime exists, handle sync
		err := a.syncExistingEntry(ctx, db, anime, entry, trackerName, localTrackingMap)
		if err != nil {
			return fmt.Errorf("failed to sync existing entry: %w", err)
		}
	}

	return nil
}

// syncExistingEntry syncs an existing anime entry between local and remote
func (a *App) syncExistingEntry(ctx context.Context, db *database.DB, anime *database.Anime, remoteEntry *tracker.UserAnimeEntry, trackerName string, localTrackingMap map[string]*database.AnimeTracking) error {
	localTracking := localTrackingMap[remoteEntry.ID]

	if localTracking == nil {
		// No local tracking, create it from remote
		tracking := &database.AnimeTracking{
			AnimeID:        anime.ID,
			Tracker:        trackerName,
			TrackerID:      remoteEntry.ID,
			Status:         string(remoteEntry.Status),
			Score:          remoteEntry.Score,
			CurrentEpisode: remoteEntry.Progress,
			TotalEpisodes:  remoteEntry.Episodes,
			LastUpdated:    remoteEntry.LastUpdated,
		}

		return db.AddAnimeTracking(tracking)
	}

	// Both local and remote exist, sync based on timestamps and progress
	if remoteEntry.LastUpdated.After(localTracking.LastUpdated) {
		// Remote is newer, update local
		tracking := &database.AnimeTracking{
			AnimeID:        anime.ID,
			Tracker:        trackerName,
			TrackerID:      remoteEntry.ID,
			Status:         string(remoteEntry.Status),
			Score:          remoteEntry.Score,
			CurrentEpisode: remoteEntry.Progress,
			TotalEpisodes:  remoteEntry.Episodes,
			LastUpdated:    remoteEntry.LastUpdated,
		}

		return db.AddAnimeTracking(tracking)
	} else if localTracking.LastUpdated.After(remoteEntry.LastUpdated) && localTracking.CurrentEpisode > remoteEntry.Progress {
		// Local is newer and has higher progress, update remote
		t, err := a.trackerMgr.GetTracker(trackerName)
		if err != nil {
			return fmt.Errorf("failed to get tracker for update: %w", err)
		}

		return t.UpdateAnimeStatus(ctx, remoteEntry.ID, tracker.Status(localTracking.Status), localTracking.CurrentEpisode, localTracking.Score)
	}
	// If remote is newer but has lower progress, or if timestamps are equal, keep local as-is

	return nil
}

// deleteLocalEntry deletes a local anime entry that was removed from remote
func (a *App) deleteLocalEntry(db *database.DB, localTracking *database.AnimeTracking, trackerName string) error {
	// Delete the tracking entry
	err := db.DeleteAnimeTracking(localTracking.AnimeID, trackerName)
	if err != nil {
		return fmt.Errorf("failed to delete tracking: %w", err)
	}

	// Check if this anime has tracking from other services
	allTrackings, err := db.GetAllAnimeTracking(localTracking.AnimeID)
	if err != nil {
		return fmt.Errorf("failed to get all tracking for anime: %w", err)
	}

	// If no other tracking exists, delete the anime entirely
	if len(allTrackings) == 0 {
		err = db.DeleteAnime(localTracking.AnimeID)
		if err != nil {
			return fmt.Errorf("failed to delete anime: %w", err)
		}
	}

	return nil
}
