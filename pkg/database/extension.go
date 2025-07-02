package database

import (
	"time"
)

// Extension represents an extension in the database
type Extension struct {
	ID            int64
	Name          string
	Package       string
	Language      string
	Version       string
	NSFW          bool
	Path          string
	RepositoryURL string
	InstalledAt   time.Time
	UpdatedAt     time.Time
}

// Source represents a source provided by an extension
type Source struct {
	ID          int64
	SourceID    string
	ExtensionID int64
	Name        string
	Language    string
	BaseURL     string
	NSFW        bool
}

// AnimeSource represents a link between an anime and a source
type AnimeSource struct {
	ID            int64
	AnimeID       int64
	SourceID      int64
	SourceAnimeID string
}

// AddExtension adds a new extension to the database
func (db *DB) AddExtension(ext *Extension) error {
	result, err := db.conn.Exec(
		`INSERT INTO extension (
			name, package, language, version, nsfw, path, repository_url,
			installed_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(package) DO UPDATE SET
			name = ?, language = ?, version = ?, nsfw = ?, path = ?, 
			repository_url = ?, updated_at = CURRENT_TIMESTAMP`,
		ext.Name, ext.Package, ext.Language, ext.Version, ext.NSFW, ext.Path, ext.RepositoryURL,
		ext.Name, ext.Language, ext.Version, ext.NSFW, ext.Path, ext.RepositoryURL,
	)
	if err != nil {
		return err
	}

	// Get the inserted ID if this was a new extension
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	if ext.ID == 0 {
		ext.ID = id
	}

	return nil
}

// GetExtensionByPackage retrieves an extension by its package name
func (db *DB) GetExtensionByPackage(pkg string) (*Extension, error) {
	var ext Extension

	err := db.conn.QueryRow(
		`SELECT 
			id, name, package, language, version, nsfw, path, repository_url,
			installed_at, updated_at
		FROM extension WHERE package = ?`, pkg,
	).Scan(
		&ext.ID, &ext.Name, &ext.Package, &ext.Language, &ext.Version, &ext.NSFW,
		&ext.Path, &ext.RepositoryURL, &ext.InstalledAt, &ext.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &ext, nil
}

// GetAllExtensions retrieves all extensions
func (db *DB) GetAllExtensions() ([]*Extension, error) {
	rows, err := db.conn.Query(
		`SELECT 
			id, name, package, language, version, nsfw, path, repository_url,
			installed_at, updated_at
		FROM extension ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var extensions []*Extension
	for rows.Next() {
		var ext Extension
		err := rows.Scan(
			&ext.ID, &ext.Name, &ext.Package, &ext.Language, &ext.Version, &ext.NSFW,
			&ext.Path, &ext.RepositoryURL, &ext.InstalledAt, &ext.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		extensions = append(extensions, &ext)
	}

	return extensions, rows.Err()
}

// DeleteExtension deletes an extension by package name
func (db *DB) DeleteExtension(pkg string) error {
	_, err := db.conn.Exec("DELETE FROM extension WHERE package = ?", pkg)
	return err
}

// AddSource adds a new source to the database
func (db *DB) AddSource(source *Source) error {
	result, err := db.conn.Exec(
		`INSERT INTO source (
			source_id, extension_id, name, language, base_url, nsfw
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_id) DO UPDATE SET
			extension_id = ?, name = ?, language = ?, base_url = ?, nsfw = ?`,
		source.SourceID, source.ExtensionID, source.Name, source.Language, source.BaseURL, source.NSFW,
		source.ExtensionID, source.Name, source.Language, source.BaseURL, source.NSFW,
	)
	if err != nil {
		return err
	}

	// Get the inserted ID if this was a new source
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	if source.ID == 0 {
		source.ID = id
	}

	return nil
}

// GetSourceByID retrieves a source by its unique ID
func (db *DB) GetSourceByID(sourceID string) (*Source, error) {
	var source Source

	err := db.conn.QueryRow(
		`SELECT 
			id, source_id, extension_id, name, language, base_url, nsfw
		FROM source WHERE source_id = ?`, sourceID,
	).Scan(
		&source.ID, &source.SourceID, &source.ExtensionID, &source.Name,
		&source.Language, &source.BaseURL, &source.NSFW,
	)
	if err != nil {
		return nil, err
	}

	return &source, nil
}

// GetSourcesByExtension retrieves all sources for a specific extension
func (db *DB) GetSourcesByExtension(extensionID int64) ([]*Source, error) {
	rows, err := db.conn.Query(
		`SELECT 
			id, source_id, extension_id, name, language, base_url, nsfw
		FROM source WHERE extension_id = ? ORDER BY name`,
		extensionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []*Source
	for rows.Next() {
		var source Source
		err := rows.Scan(
			&source.ID, &source.SourceID, &source.ExtensionID, &source.Name,
			&source.Language, &source.BaseURL, &source.NSFW,
		)
		if err != nil {
			return nil, err
		}

		sources = append(sources, &source)
	}

	return sources, rows.Err()
}

// GetAllSources retrieves all sources
func (db *DB) GetAllSources() ([]*Source, error) {
	rows, err := db.conn.Query(
		`SELECT 
			id, source_id, extension_id, name, language, base_url, nsfw
		FROM source ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []*Source
	for rows.Next() {
		var source Source
		err := rows.Scan(
			&source.ID, &source.SourceID, &source.ExtensionID, &source.Name,
			&source.Language, &source.BaseURL, &source.NSFW,
		)
		if err != nil {
			return nil, err
		}

		sources = append(sources, &source)
	}

	return sources, rows.Err()
}

// DeleteSource deletes a source by its unique ID
func (db *DB) DeleteSource(sourceID string) error {
	_, err := db.conn.Exec("DELETE FROM source WHERE source_id = ?", sourceID)
	return err
}

// AddAnimeSource links an anime to a source
func (db *DB) AddAnimeSource(animeSource *AnimeSource) error {
	result, err := db.conn.Exec(
		`INSERT INTO anime_source (
			anime_id, source_id, source_anime_id
		) VALUES (?, ?, ?)
		ON CONFLICT(anime_id, source_id) DO UPDATE SET
			source_anime_id = ?`,
		animeSource.AnimeID, animeSource.SourceID, animeSource.SourceAnimeID,
		animeSource.SourceAnimeID,
	)
	if err != nil {
		return err
	}

	// Get the inserted ID if this was a new link
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	if animeSource.ID == 0 {
		animeSource.ID = id
	}

	return nil
}

// GetAnimeSources retrieves all sources for a specific anime
func (db *DB) GetAnimeSources(animeID int64) ([]*AnimeSource, error) {
	rows, err := db.conn.Query(
		`SELECT 
			id, anime_id, source_id, source_anime_id
		FROM anime_source WHERE anime_id = ?`,
		animeID,
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

// GetAnimeSourceBySourceAnimeID retrieves an anime source by source anime ID and source ID
func (db *DB) GetAnimeSourceBySourceAnimeID(sourceID int64, sourceAnimeID string) (*AnimeSource, error) {
	var source AnimeSource
	err := db.conn.QueryRow(
		`SELECT 
			id, anime_id, source_id, source_anime_id
		FROM anime_source WHERE source_id = ? AND source_anime_id = ?`,
		sourceID, sourceAnimeID,
	).Scan(
		&source.ID, &source.AnimeID, &source.SourceID, &source.SourceAnimeID,
	)
	if err != nil {
		return nil, err
	}

	return &source, nil
}

// DeleteAnimeSource removes the link between an anime and a source
func (db *DB) DeleteAnimeSource(animeID int64, sourceID int64) error {
	_, err := db.conn.Exec(
		"DELETE FROM anime_source WHERE anime_id = ? AND source_id = ?",
		animeID, sourceID,
	)
	return err
}
