package tracker

import (
	"context"
	"fmt"
	"time"

	"github.com/wraient/pair/pkg/database"
)

// SyncManager handles automatic synchronization between local database and external trackers
type SyncManager struct {
	db        *database.DB
	manager   *TrackerManager
	isRunning bool
	stopCh    chan struct{}
}

// NewSyncManager creates a new SyncManager
func NewSyncManager(db *database.DB, manager *TrackerManager) *SyncManager {
	return &SyncManager{
		db:      db,
		manager: manager,
		stopCh:  make(chan struct{}),
	}
}

// Start starts the automatic synchronization process
func (s *SyncManager) Start() {
	if s.isRunning {
		return
	}

	s.isRunning = true
	go s.syncLoop()
}

// Stop stops the automatic synchronization process
func (s *SyncManager) Stop() {
	if !s.isRunning {
		return
	}

	s.stopCh <- struct{}{}
	s.isRunning = false
}

// syncLoop periodically syncs data with external trackers
func (s *SyncManager) syncLoop() {
	// Get sync interval from config
	syncInterval := 60 // Default to 60 minutes
	intervalStr, err := s.db.GetConfig("tracker_sync_interval")
	if err == nil && intervalStr != "" {
		fmt.Sscanf(intervalStr, "%d", &syncInterval)
	}

	// Minimum interval is 15 minutes to avoid rate limiting
	if syncInterval < 15 {
		syncInterval = 15
	}

	ticker := time.NewTicker(time.Duration(syncInterval) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performSync()
		case <-s.stopCh:
			return
		}
	}
}

// performSync performs synchronization with all trackers
func (s *SyncManager) performSync() {
	// Check if auto sync is enabled
	autoSyncStr, err := s.db.GetConfig("tracker_auto_sync")
	if err != nil || autoSyncStr != "true" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Get trackers that should be synced
	trackers := []string{"mal", "anilist"}
	for _, name := range trackers {
		tracker, err := s.manager.GetTracker(name)
		if err != nil {
			continue
		}

		// Skip if not authenticated
		if !tracker.IsAuthenticated() {
			continue
		}

		// Sync from tracker to local
		_, err = tracker.SyncFromRemote(ctx, s.db)
		if err != nil {
			fmt.Printf("Error syncing from %s: %v\n", name, err)
		}

		// Sync from local to tracker
		_, err = tracker.SyncToRemote(ctx, s.db)
		if err != nil {
			fmt.Printf("Error syncing to %s: %v\n", name, err)
		}
	}
}

// SyncEpisodeProgress syncs episode progress to all trackers
func (s *SyncManager) SyncEpisodeProgress(animeID int64, episodeNumber float64) error {
	// Get anime tracking entries
	trackings, err := s.db.GetAllAnimeTrackingByAnimeID(animeID)
	if err != nil {
		return fmt.Errorf("failed to get anime tracking entries: %w", err)
	}

	if len(trackings) == 0 {
		return nil // No tracking entries to sync
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, tracking := range trackings {
		// Skip local tracker
		if tracking.Tracker == "local" {
			continue
		}

		// Skip if not authenticated
		tracker, err := s.manager.GetTracker(tracking.Tracker)
		if err != nil {
			continue
		}

		if !tracker.IsAuthenticated() {
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

		// Update episode progress on tracker
		err = tracker.UpdateAnimeStatus(ctx, tracking.TrackerID, status, episodeNumber, tracking.Score)
		if err != nil {
			fmt.Printf("Error updating progress on %s: %v\n", tracking.Tracker, err)
		}

		// Update local tracking record
		tracking.CurrentEpisode = episodeNumber
		tracking.LastUpdated = time.Now()
		err = s.db.UpdateAnimeTracking(tracking.AnimeID, tracking.Tracker, episodeNumber)
		if err != nil {
			fmt.Printf("Error updating local tracking: %v\n", err)
		}
	}

	return nil
}
