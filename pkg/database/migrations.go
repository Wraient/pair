package database

import (
	"database/sql"
	"fmt"
	"sort"
	"time"
)

// Migration represents a database migration
type Migration struct {
	Version     int
	Description string
	SQL         string
}

// RunMigrations applies any pending migrations to the database
func (db *DB) RunMigrations(migrations []Migration) error {
	// Create migrations table if it doesn't exist
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			version INTEGER PRIMARY KEY,
			description TEXT NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get the current database version
	var currentVersion int
	err = db.conn.QueryRow("SELECT COALESCE(MAX(version), 0) FROM migrations").Scan(&currentVersion)
	if err != nil {
		return fmt.Errorf("failed to get current database version: %w", err)
	}

	// Sort migrations by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	// Apply pending migrations
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		// Start a transaction for this migration
		tx, err := db.conn.Begin()
		if err != nil {
			return fmt.Errorf("failed to start transaction for migration %d: %w", migration.Version, err)
		}

		// Apply the migration
		_, err = tx.Exec(migration.SQL)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}

		// Record the migration
		_, err = tx.Exec(
			"INSERT INTO migrations (version, description, applied_at) VALUES (?, ?, ?)",
			migration.Version, migration.Description, time.Now(),
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		// Commit the transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
		}

		fmt.Printf("Applied migration %d: %s\n", migration.Version, migration.Description)
	}

	return nil
}

// GetDatabaseVersion returns the current database schema version
func (db *DB) GetDatabaseVersion() (int, error) {
	var version int
	err := db.conn.QueryRow("SELECT COALESCE(MAX(version), 0) FROM migrations").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get database version: %w", err)
	}
	return version, nil
}

// BackupDatabase creates a backup of the database
func (db *DB) BackupDatabase(backupPath string) error {
	// Get a connection to the backup database
	backup, err := sql.Open("sqlite3", backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup database: %w", err)
	}
	defer backup.Close()

	// Start a transaction
	tx, err := backup.Begin()
	if err != nil {
		return fmt.Errorf("failed to start backup transaction: %w", err)
	}
	defer tx.Rollback()

	// Export schema and data
	_, err = tx.Exec("VACUUM INTO ?", backupPath)
	if err != nil {
		return fmt.Errorf("failed to backup database: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit backup: %w", err)
	}

	return nil
}

// InitialMigration returns the initial migration that creates the schema
func InitialMigration() Migration {
	return Migration{
		Version:     1,
		Description: "Initial schema",
		SQL: `
			-- Config table
			CREATE TABLE IF NOT EXISTS config (
				key TEXT PRIMARY KEY,
				value TEXT NOT NULL,
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			);

			-- Anime table
			CREATE TABLE IF NOT EXISTS anime (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				title TEXT NOT NULL,
				original_title TEXT,
				alternative_titles TEXT, -- JSON array of alternative titles
				description TEXT,
				total_episodes INTEGER,
				type TEXT, -- TV, Movie, OVA, etc.
				year INTEGER,
				season TEXT, -- Winter, Spring, Summer, Fall
				status TEXT, -- Airing, Completed, etc.
				genres TEXT, -- JSON array of genres
				thumbnail_url TEXT,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			);

			-- Create index on anime title for faster searches
			CREATE INDEX IF NOT EXISTS idx_anime_title ON anime(title);

			-- AnimeTracking table
			CREATE TABLE IF NOT EXISTS anime_tracking (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				anime_id INTEGER NOT NULL,
				tracker TEXT NOT NULL, -- local, anilist, mal, etc.
				tracker_id TEXT, -- ID in the external tracker system
				status TEXT NOT NULL, -- watching, completed, on_hold, dropped, plan_to_watch
				score REAL, -- User rating
				current_episode REAL, -- Supporting fractional episodes (e.g., 12.5)
				total_episodes INTEGER,
				last_updated TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (anime_id) REFERENCES anime(id) ON DELETE CASCADE,
				UNIQUE (anime_id, tracker)
			);

			-- EpisodeProgress table
			CREATE TABLE IF NOT EXISTS episode_progress (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				anime_id INTEGER NOT NULL,
				episode_number REAL NOT NULL, -- Supporting fractional episodes (e.g., 12.5)
				position INTEGER NOT NULL, -- Position in seconds
				duration INTEGER NOT NULL, -- Duration in seconds
				playback_speed REAL NOT NULL DEFAULT 1.0, -- Last used playback speed
				watched BOOLEAN NOT NULL DEFAULT 0, -- Whether the episode has been watched completely
				source_id TEXT, -- Source ID used to watch this episode
				last_watched TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (anime_id) REFERENCES anime(id) ON DELETE CASCADE,
				UNIQUE (anime_id, episode_number)
			);

			-- Extension table
			CREATE TABLE IF NOT EXISTS extension (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL,
				package TEXT NOT NULL,
				language TEXT NOT NULL,
				version TEXT NOT NULL,
				nsfw BOOLEAN NOT NULL DEFAULT 0,
				path TEXT NOT NULL, -- Path to the extension binary
				repository_url TEXT, -- URL to the repository where the extension was installed from
				installed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				UNIQUE (package)
			);

			-- Source table
			CREATE TABLE IF NOT EXISTS source (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				source_id TEXT NOT NULL, -- Unique ID for the source
				extension_id INTEGER NOT NULL,
				name TEXT NOT NULL,
				language TEXT NOT NULL,
				base_url TEXT NOT NULL,
				nsfw BOOLEAN NOT NULL DEFAULT 0,
				FOREIGN KEY (extension_id) REFERENCES extension(id) ON DELETE CASCADE,
				UNIQUE (source_id)
			);

			-- AnimeSource table
			CREATE TABLE IF NOT EXISTS anime_source (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				anime_id INTEGER NOT NULL,
				source_id INTEGER NOT NULL,
				source_anime_id TEXT NOT NULL, -- ID of the anime in the source
				FOREIGN KEY (anime_id) REFERENCES anime(id) ON DELETE CASCADE,
				FOREIGN KEY (source_id) REFERENCES source(id) ON DELETE CASCADE,
				UNIQUE (anime_id, source_id)
			);

			-- Episode table
			CREATE TABLE IF NOT EXISTS episode (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				anime_id INTEGER NOT NULL,
				number REAL NOT NULL, -- Supporting fractional episodes (e.g., 12.5)
				title TEXT,
				description TEXT,
				duration INTEGER, -- Duration in seconds
				thumbnail_url TEXT,
				air_date TEXT,
				is_filler BOOLEAN NOT NULL DEFAULT 0,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (anime_id) REFERENCES anime(id) ON DELETE CASCADE,
				UNIQUE (anime_id, number)
			);
		`,
	}
}
