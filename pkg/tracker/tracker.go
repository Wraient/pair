package tracker

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/wraient/pair/pkg/database"
)

// Status represents the watch status of an anime
type Status string

// Common anime watch statuses
const (
	StatusWatching    Status = "watching"
	StatusCompleted   Status = "completed"
	StatusOnHold      Status = "on_hold"
	StatusDropped     Status = "dropped"
	StatusPlanToWatch Status = "plan_to_watch"
)

// Tracker is the interface that must be implemented by all trackers
type Tracker interface {
	// Name returns the name of the tracker
	Name() string

	// IsAuthenticated checks if the user is authenticated with the tracker
	IsAuthenticated() bool

	// Authenticate authenticates the user with the tracker
	Authenticate(ctx context.Context) error

	// SearchAnime searches for anime on the tracker
	SearchAnime(ctx context.Context, query string, limit int) ([]AnimeInfo, error)

	// GetAnimeDetails gets detailed information about an anime
	GetAnimeDetails(ctx context.Context, id string) (*AnimeInfo, error)

	// GetUserAnimeList gets the user's anime list
	GetUserAnimeList(ctx context.Context) ([]UserAnimeEntry, error)

	// UpdateAnimeStatus updates the watch status of an anime
	UpdateAnimeStatus(ctx context.Context, id string, status Status, episode float64, score float64) error

	// SyncFromRemote synchronizes the local database with the remote tracker
	SyncFromRemote(ctx context.Context, db *database.DB) (SyncStats, error)

	// SyncToRemote synchronizes the remote tracker with the local database
	SyncToRemote(ctx context.Context, db *database.DB) (SyncStats, error)
}

// AnimeInfo represents basic anime information from a tracker
type AnimeInfo struct {
	ID                string
	Title             string
	EnglishTitle      string
	JapaneseTitle     string
	AlternativeTitles []string
	Synopsis          string
	Type              string
	Status            string
	Episodes          int
	StartDate         time.Time
	EndDate           time.Time
	Season            string
	Year              int
	Rating            float64
	Genres            []string
	Studios           []string
	ImageURL          string
}

// UserAnimeEntry represents an entry in a user's anime list
type UserAnimeEntry struct {
	AnimeInfo
	Status      Status
	Score       float64
	Progress    float64
	StartDate   time.Time
	EndDate     time.Time
	Notes       string
	LastUpdated time.Time
}

// SyncStats contains statistics about a sync operation
type SyncStats struct {
	Added   int
	Updated int
	Deleted int
	Skipped int
	Errors  int
	Details []string
}

// TrackerManager manages all trackers
type TrackerManager struct {
	trackers map[string]Tracker
	db       *database.DB
}

// NewTrackerManager creates a new TrackerManager
func NewTrackerManager(db *database.DB) *TrackerManager {
	return &TrackerManager{
		trackers: make(map[string]Tracker),
		db:       db,
	}
}

// RegisterTracker registers a tracker with the manager
func (m *TrackerManager) RegisterTracker(tracker Tracker) {
	m.trackers[tracker.Name()] = tracker
}

// GetTracker returns a tracker by name
func (m *TrackerManager) GetTracker(name string) (Tracker, error) {
	tracker, ok := m.trackers[name]
	if !ok {
		return nil, fmt.Errorf("tracker %q not found", name)
	}
	return tracker, nil
}

// GetActiveTracker returns the currently active tracker
func (m *TrackerManager) GetActiveTracker() (Tracker, error) {
	// Get active tracker from config
	activeTrackerName, err := m.db.GetConfig("active_tracker")
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Default to local tracker if not set
			activeTrackerName = "local"
			if err := m.db.SetConfig("active_tracker", "local"); err != nil {
				return nil, fmt.Errorf("failed to set default active tracker: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to get active tracker from config: %w", err)
		}
	}

	return m.GetTracker(activeTrackerName)
}

// SetActiveTracker sets the active tracker
func (m *TrackerManager) SetActiveTracker(name string) error {
	// Check if tracker exists
	if _, ok := m.trackers[name]; !ok {
		return fmt.Errorf("tracker %q not found", name)
	}

	// Set active tracker in config
	return m.db.SetConfig("active_tracker", name)
}

// SyncAllFromRemote synchronizes all trackers from remote to local
func (m *TrackerManager) SyncAllFromRemote(ctx context.Context) (map[string]SyncStats, error) {
	stats := make(map[string]SyncStats)

	for name, tracker := range m.trackers {
		if !tracker.IsAuthenticated() {
			continue
		}

		syncStats, err := tracker.SyncFromRemote(ctx, m.db)
		if err != nil {
			return stats, fmt.Errorf("failed to sync from %s: %w", name, err)
		}

		stats[name] = syncStats
	}

	return stats, nil
}

// SyncAllToRemote synchronizes all trackers from local to remote
func (m *TrackerManager) SyncAllToRemote(ctx context.Context) (map[string]SyncStats, error) {
	stats := make(map[string]SyncStats)

	for name, tracker := range m.trackers {
		if !tracker.IsAuthenticated() {
			continue
		}

		syncStats, err := tracker.SyncToRemote(ctx, m.db)
		if err != nil {
			return stats, fmt.Errorf("failed to sync to %s: %w", name, err)
		}

		stats[name] = syncStats
	}

	return stats, nil
}
