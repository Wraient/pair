package database

import (
	"os"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	// Create a temporary file for the test database
	tmpfile, err := os.CreateTemp("", "pair-test-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpfile.Close()

	// Initialize the database
	db, err := New(tmpfile.Name())
	if err != nil {
		os.Remove(tmpfile.Name())
		t.Fatalf("Failed to initialize database: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		db.Close()
		os.Remove(tmpfile.Name())
	}

	return db, cleanup
}

func TestConfigOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Test setting a config value
	if err := db.SetConfig("test_key", "test_value"); err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Test getting a config value
	value, err := db.GetConfig("test_key")
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}
	if value != "test_value" {
		t.Errorf("Expected config value 'test_value', got '%s'", value)
	}

	// Test getting a non-existent config value
	value, err = db.GetConfig("nonexistent_key")
	if err != nil {
		t.Fatalf("Failed to get nonexistent config: %v", err)
	}
	if value != "" {
		t.Errorf("Expected empty string for nonexistent key, got '%s'", value)
	}

	// Test getting all config entries
	entries, err := db.GetAllConfig()
	if err != nil {
		t.Fatalf("Failed to get all config entries: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("Expected 1 config entry, got %d", len(entries))
	}

	// Test deleting a config value
	if err := db.DeleteConfig("test_key"); err != nil {
		t.Fatalf("Failed to delete config: %v", err)
	}

	// Verify deletion
	value, err = db.GetConfig("test_key")
	if err != nil {
		t.Fatalf("Failed to get deleted config: %v", err)
	}
	if value != "" {
		t.Errorf("Expected empty string after deletion, got '%s'", value)
	}
}

func TestAnimeOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a test anime
	anime := &Anime{
		Title:             "Test Anime",
		OriginalTitle:     "テストアニメ",
		AlternativeTitles: []string{"Test Show", "Testing Anime"},
		Description:       "This is a test anime",
		TotalEpisodes:     12,
		Type:              "TV",
		Year:              2025,
		Season:            "Spring",
		Status:            "Airing",
		Genres:            []string{"Action", "Comedy"},
		ThumbnailURL:      "https://example.com/image.jpg",
	}

	// Test adding an anime
	if err := db.AddAnime(anime); err != nil {
		t.Fatalf("Failed to add anime: %v", err)
	}
	if anime.ID == 0 {
		t.Errorf("Expected non-zero ID after adding anime")
	}

	// Test getting anime by ID
	retrievedAnime, err := db.GetAnimeByID(anime.ID)
	if err != nil {
		t.Fatalf("Failed to get anime by ID: %v", err)
	}
	if retrievedAnime.Title != anime.Title {
		t.Errorf("Expected title '%s', got '%s'", anime.Title, retrievedAnime.Title)
	}

	// Test getting anime by title
	retrievedAnime, err = db.GetAnimeByTitle("Test Anime")
	if err != nil {
		t.Fatalf("Failed to get anime by title: %v", err)
	}
	if retrievedAnime.ID != anime.ID {
		t.Errorf("Expected ID %d, got %d", anime.ID, retrievedAnime.ID)
	}

	// Test searching for anime
	results, err := db.SearchAnime("Test")
	if err != nil {
		t.Fatalf("Failed to search for anime: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 search result, got %d", len(results))
	}

	// Test updating anime
	anime.Description = "Updated description"
	if err := db.UpdateAnime(anime); err != nil {
		t.Fatalf("Failed to update anime: %v", err)
	}

	// Verify update
	retrievedAnime, err = db.GetAnimeByID(anime.ID)
	if err != nil {
		t.Fatalf("Failed to get updated anime: %v", err)
	}
	if retrievedAnime.Description != "Updated description" {
		t.Errorf("Expected updated description, got '%s'", retrievedAnime.Description)
	}

	// Test deleting anime
	if err := db.DeleteAnime(anime.ID); err != nil {
		t.Fatalf("Failed to delete anime: %v", err)
	}

	// Verify deletion
	_, err = db.GetAnimeByID(anime.ID)
	if err == nil {
		t.Errorf("Expected error after deleting anime")
	}
}

func TestAnimeTrackingOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a test anime
	anime := &Anime{
		Title:         "Test Anime",
		TotalEpisodes: 24,
	}
	if err := db.AddAnime(anime); err != nil {
		t.Fatalf("Failed to add anime: %v", err)
	}

	// Create tracking info
	tracking := &AnimeTracking{
		AnimeID:        anime.ID,
		Tracker:        "local",
		Status:         "watching",
		Score:          8.5,
		CurrentEpisode: 12.0,
		TotalEpisodes:  24,
	}

	// Test adding tracking info
	if err := db.AddAnimeTracking(tracking); err != nil {
		t.Fatalf("Failed to add tracking info: %v", err)
	}
	if tracking.ID == 0 {
		t.Errorf("Expected non-zero ID after adding tracking info")
	}

	// Test getting tracking info
	retrievedTracking, err := db.GetAnimeTracking(anime.ID, "local")
	if err != nil {
		t.Fatalf("Failed to get tracking info: %v", err)
	}
	if retrievedTracking.CurrentEpisode != 12.0 {
		t.Errorf("Expected current episode 12.0, got %f", retrievedTracking.CurrentEpisode)
	}

	// Test updating tracking info
	err = db.UpdateAnimeTracking(anime.ID, "local", 13.0)
	if err != nil {
		t.Fatalf("Failed to update tracking info: %v", err)
	}

	// Verify update
	retrievedTracking, err = db.GetAnimeTracking(anime.ID, "local")
	if err != nil {
		t.Fatalf("Failed to get updated tracking info: %v", err)
	}
	if retrievedTracking.CurrentEpisode != 13.0 {
		t.Errorf("Expected updated current episode 13.0, got %f", retrievedTracking.CurrentEpisode)
	}
}

func TestEpisodeOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a test anime
	anime := &Anime{
		Title:         "Test Anime",
		TotalEpisodes: 12,
	}
	if err := db.AddAnime(anime); err != nil {
		t.Fatalf("Failed to add anime: %v", err)
	}

	// Create an episode
	episode := &Episode{
		AnimeID:      anime.ID,
		Number:       1.0,
		Title:        "Episode 1",
		Description:  "First episode",
		ThumbnailURL: "https://example.com/ep1.jpg",
	}

	// Test adding an episode
	if err := db.AddEpisode(episode); err != nil {
		t.Fatalf("Failed to add episode: %v", err)
	}
	if episode.ID == 0 {
		t.Errorf("Expected non-zero ID after adding episode")
	}

	// Create episode progress
	progress := &EpisodeProgress{
		AnimeID:       anime.ID,
		EpisodeNumber: 1.0,
		Position:      300,
		Duration:      1440,
		PlaybackSpeed: 1.0,
		SourceID:      "test_source",
		LastWatched:   time.Now(),
	}

	// Test adding episode progress
	if err := db.AddEpisodeProgress(progress); err != nil {
		t.Fatalf("Failed to add episode progress: %v", err)
	}
	if progress.ID == 0 {
		t.Errorf("Expected non-zero ID after adding episode progress")
	}
}

func TestExtensionOperations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a test extension
	ext := &Extension{
		Name:          "Test Extension",
		Package:       "test-extension",
		Language:      "en",
		Version:       "1.0.0",
		NSFW:          false,
		Path:          "/path/to/extension",
		RepositoryURL: "https://github.com/user/test-extension",
	}

	// Test adding an extension
	if err := db.AddExtension(ext); err != nil {
		t.Fatalf("Failed to add extension: %v", err)
	}
	if ext.ID == 0 {
		t.Errorf("Expected non-zero ID after adding extension")
	}

	// Test getting an extension
	retrievedExt, err := db.GetExtensionByPackage("test-extension")
	if err != nil {
		t.Fatalf("Failed to get extension: %v", err)
	}
	if retrievedExt.Name != "Test Extension" {
		t.Errorf("Expected name 'Test Extension', got '%s'", retrievedExt.Name)
	}

	// Test getting all extensions
	extensions, err := db.GetAllExtensions()
	if err != nil {
		t.Fatalf("Failed to get all extensions: %v", err)
	}
	if len(extensions) != 1 {
		t.Errorf("Expected 1 extension, got %d", len(extensions))
	}

	// Create a test source
	source := &Source{
		SourceID:    "test-source",
		ExtensionID: ext.ID,
		Name:        "Test Source",
		Language:    "en",
		BaseURL:     "https://example.com",
		NSFW:        false,
	}

	// Test adding a source
	if err := db.AddSource(source); err != nil {
		t.Fatalf("Failed to add source: %v", err)
	}
	if source.ID == 0 {
		t.Errorf("Expected non-zero ID after adding source")
	}

	// Test getting a source
	retrievedSource, err := db.GetSourceByID("test-source")
	if err != nil {
		t.Fatalf("Failed to get source: %v", err)
	}
	if retrievedSource.Name != "Test Source" {
		t.Errorf("Expected name 'Test Source', got '%s'", retrievedSource.Name)
	}

	// Test deleting an extension
	if err := db.DeleteExtension("test-extension"); err != nil {
		t.Fatalf("Failed to delete extension: %v", err)
	}

	// Verify extension deletion
	_, err = db.GetExtensionByPackage("test-extension")
	if err == nil {
		t.Errorf("Expected error after deleting extension")
	}

	// Verify source deletion (should be cascaded)
	_, err = db.GetSourceByID("test-source")
	if err == nil {
		t.Errorf("Expected error after deleting extension (cascade to source)")
	}
}

func TestMigrations(t *testing.T) {
	// Create a temporary file for the test database
	tmpfile, err := os.CreateTemp("", "pair-test-migrations-*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpfile.Close()
	defer os.Remove(tmpfile.Name())

	// Initialize the database
	db, err := New(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Get current version
	version, err := db.GetDatabaseVersion()
	if err != nil {
		t.Fatalf("Failed to get database version: %v", err)
	}
	if version < 1 {
		t.Errorf("Expected database version to be at least 1, got %d", version)
	}

	// Test applying a new migration
	newMigration := Migration{
		Version:     version + 1,
		Description: "Test migration",
		SQL: `
			CREATE TABLE test_table (
				id INTEGER PRIMARY KEY,
				name TEXT NOT NULL
			);
		`,
	}

	if err := db.RunMigrations([]Migration{newMigration}); err != nil {
		t.Fatalf("Failed to run migration: %v", err)
	}

	// Verify new version
	newVersion, err := db.GetDatabaseVersion()
	if err != nil {
		t.Fatalf("Failed to get database version: %v", err)
	}
	if newVersion != version+1 {
		t.Errorf("Expected database version to be %d, got %d", version+1, newVersion)
	}

	// Test table exists
	_, err = db.conn.Exec("INSERT INTO test_table (name) VALUES ('test')")
	if err != nil {
		t.Errorf("Failed to insert into migrated table: %v", err)
	}
}
