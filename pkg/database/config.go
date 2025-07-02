package database

import (
	"database/sql"
	"time"
)

// ConfigEntry represents a configuration entry in the database
type ConfigEntry struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}

// GetConfig retrieves a configuration value
func (db *DB) GetConfig(key string) (string, error) {
	var value string
	err := db.conn.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil // No value set
	}
	return value, err
}

// SetConfig sets a configuration value
func (db *DB) SetConfig(key, value string) error {
	_, err := db.conn.Exec(
		`INSERT INTO config (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = CURRENT_TIMESTAMP`,
		key, value, value,
	)
	return err
}

// GetAllConfig retrieves all configuration entries
func (db *DB) GetAllConfig() ([]ConfigEntry, error) {
	rows, err := db.conn.Query("SELECT key, value, updated_at FROM config")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []ConfigEntry
	for rows.Next() {
		var entry ConfigEntry
		if err := rows.Scan(&entry.Key, &entry.Value, &entry.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// DeleteConfig deletes a configuration entry
func (db *DB) DeleteConfig(key string) error {
	_, err := db.conn.Exec("DELETE FROM config WHERE key = ?", key)
	return err
}
