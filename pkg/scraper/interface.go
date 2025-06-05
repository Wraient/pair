package scraper

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// Status enum for anime sources
const (
	StatusUnknown            = "unknown"
	StatusOngoing            = "ongoing"
	StatusCompleted          = "completed"
	StatusLicensed           = "licensed"
	StatusPublishingFinished = "publishing_finished"
	StatusCancelled          = "cancelled"
	StatusOnHiatus           = "on_hiatus"
)

// Anime represents an anime series
type Anime struct {
	ID                string   `json:"anime_id"`                     // URL or unique identifier for the anime
	Title             string   `json:"title"`                        // Primary title
	Artist            string   `json:"artist"`                       // Animation studio or artist
	Author            string   `json:"author"`                       // Original author/creator
	Description       string   `json:"description"`                  // Series description
	Genre             string   `json:"genre"`                        // Comma-separated genres
	ThumbnailURL      string   `json:"thumbnail_url"`                // URL to thumbnail image
	Status            string   `json:"status"`                       // Airing status (using constants above)
	AlternativeTitles []string `json:"alternative_titles,omitempty"` // List of alternative titles
	Episodes          int      `json:"episodes,omitempty"`           // Total number of episodes
	SubDub            string   `json:"sub_dub,omitempty"`            // "sub", "dub", or "both"
	Tags              []string `json:"tags,omitempty"`               // List of tags
	ReleaseYear       int      `json:"release_year,omitempty"`       // Year of release
}

// Episode represents a single episode of an anime
type Episode struct {
	ID            string  `json:"anime_id"`            // URL or unique identifier for the episode
	Name          string  `json:"name"`                // Episode title
	DateUpload    int64   `json:"date_upload"`         // Unix timestamp of upload date
	EpisodeNumber float64 `json:"episode_number"`      // Episode number (supports 1.5, etc.)
	Scanlator     string  `json:"scanlator,omitempty"` // Subtitle group/scanlator
}

// Track represents a subtitle or audio track
type Track struct {
	URL  string `json:"url"`  // URL to the track
	Lang string `json:"lang"` // Language code
}

// Video represents a video stream with quality information
type Video struct {
	ID            string            `json:"anime_id"`                // URL or unique identifier
	Quality       string            `json:"quality"`                 // Quality label (1080p, 720p, etc.)
	VideoURL      string            `json:"videourl"`                // Direct stream URL
	Headers       map[string]string `json:"headers,omitempty"`       // HTTP headers needed for access
	SubtitleTrack *Track            `json:"subtitleTrack,omitempty"` // Default subtitle track
	AudioTracks   []Track           `json:"audioTracks,omitempty"`   // Available audio tracks
}

// AnimePage represents a paginated result of animes
type AnimePage struct {
	Animes      []Anime `json:"animes"`      // List of anime results
	HasNextPage bool    `json:"hasNextPage"` // Whether there are more pages
}

// VideoResponse represents a response containing video streams and subtitles
type VideoResponse struct {
	Streams   []Video `json:"streams"`   // Available video streams
	Subtitles []Track `json:"subtitles"` // Available subtitle tracks
}

// MagnetResponse represents a response containing a magnet link for torrent sources
type MagnetResponse struct {
	MagnetLink string `json:"magnetLink"` // Magnet URI for torrents
}

// FilterItem represents a filter option in the filter list
type FilterItem struct {
	Type          string        `json:"type"`                    // Type of filter (header, group, select, checkbox)
	Name          string        `json:"name,omitempty"`          // Name of the filter
	Text          string        `json:"text,omitempty"`          // Display text for headers
	Entries       []FilterEntry `json:"entries,omitempty"`       // For group type
	Options       []string      `json:"options,omitempty"`       // For select type
	SelectedValue string        `json:"selectedValue,omitempty"` // For select type
	State         bool          `json:"state,omitempty"`         // For checkbox type
}

// FilterEntry represents an individual option in a filter group
type FilterEntry struct {
	Name  string `json:"name"`  // Name of the entry
	State bool   `json:"state"` // Whether it's selected
}

// FilterResponse represents the available filters for a source
type FilterResponse struct {
	Filters []FilterItem `json:"filters"` // Available filters
}

// CLIOutput represents the expected JSON output format from scraper CLIs
type CLIOutput struct {
	Status  string      `json:"status"`            // "success" or "error"
	Data    interface{} `json:"data,omitempty"`    // Contains the actual result data
	Error   string      `json:"error,omitempty"`   // Error message if status is "error"
	Message string      `json:"message,omitempty"` // Optional informational message
}

// ExtensionInfo represents metadata about an extension
type ExtensionInfo struct {
	Name    string       `json:"name"`    // Extension name
	Package string       `json:"pkg"`     // Package name
	Lang    string       `json:"lang"`    // Language code
	Version string       `json:"version"` // Version string
	NSFW    bool         `json:"nsfw"`    // Whether content is NSFW
	Sources []SourceInfo `json:"sources"` // Sources provided by this extension
}

// SourceInfo represents metadata about a source
type SourceInfo struct {
	ID                   string `json:"id"`                   // Source ID
	Name                 string `json:"name"`                 // Source name
	BaseURL              string `json:"baseUrl"`              // Base URL for the source
	Language             string `json:"language"`             // Language code
	NSFW                 bool   `json:"nsfw"`                 // Whether content is NSFW
	RateLimit            int    `json:"ratelimit"`            // Rate limit per minute
	SupportsLatest       bool   `json:"supportsLatest"`       // Whether source supports latest updates
	SupportsSearch       bool   `json:"supportsSearch"`       // Whether source supports search
	SupportsRelatedAnime bool   `json:"supportsRelatedAnime"` // Whether source supports related anime
}

// CLIScraper implements scraping functionality using the CLI tool interface
type CLIScraper struct {
	BinaryPath string
	SourceID   string
}

// NewCLIScraper creates a new CLI-based scraper
func NewCLIScraper(extensionPath string, sourceID string) *CLIScraper {
	return &CLIScraper{
		BinaryPath: extensionPath,
		SourceID:   sourceID,
	}
}

// runCommand executes a command and parses the JSON output
func (c *CLIScraper) runCommand(args ...string) (CLIOutput, error) {
	var output CLIOutput

	cmd := exec.Command(c.BinaryPath, args...)

	stdout, err := cmd.Output()
	if err != nil {
		// Try to extract stderr if available
		if exitErr, ok := err.(*exec.ExitError); ok {
			return output, fmt.Errorf("command failed: %s, stderr: %s", err, exitErr.Stderr)
		}
		return output, fmt.Errorf("command failed: %s", err)
	}

	if err := json.Unmarshal(stdout, &output); err != nil {
		return output, fmt.Errorf("failed to parse output: %s", err)
	}

	if output.Status == "error" {
		return output, fmt.Errorf("command returned error: %s", output.Error)
	}

	return output, nil
}

// GetExtensionInfo retrieves metadata about the extension
func (c *CLIScraper) GetExtensionInfo() (ExtensionInfo, error) {
	var info ExtensionInfo

	output, err := c.runCommand("extension-info")
	if err != nil {
		return info, err
	}

	// Convert the data to JSON and then unmarshal to our struct
	data, err := json.Marshal(output.Data)
	if err != nil {
		return info, fmt.Errorf("failed to re-marshal data: %s", err)
	}

	if err := json.Unmarshal(data, &info); err != nil {
		return info, fmt.Errorf("failed to unmarshal extension info: %s", err)
	}

	return info, nil
}

// GetSourceInfo retrieves metadata about a specific source
func (c *CLIScraper) GetSourceInfo() (SourceInfo, error) {
	var info SourceInfo

	output, err := c.runCommand("source-info", c.SourceID)
	if err != nil {
		return info, err
	}

	// Convert the data to JSON and then unmarshal to our struct
	data, err := json.Marshal(output.Data)
	if err != nil {
		return info, fmt.Errorf("failed to re-marshal data: %s", err)
	}

	if err := json.Unmarshal(data, &info); err != nil {
		return info, fmt.Errorf("failed to unmarshal source info: %s", err)
	}

	return info, nil
}

// GetPopularAnime retrieves popular anime from the source
func (c *CLIScraper) GetPopularAnime(page int) ([]Anime, error) {
	var animes []Anime

	output, err := c.runCommand("popular", c.SourceID, "--page", strconv.Itoa(page))
	if err != nil {
		return animes, err
	}

	// Convert the data to JSON and then unmarshal to our struct
	data, err := json.Marshal(output.Data)
	if err != nil {
		return animes, fmt.Errorf("failed to re-marshal data: %s", err)
	}

	if err := json.Unmarshal(data, &animes); err != nil {
		return animes, fmt.Errorf("failed to unmarshal anime list: %s", err)
	}

	return animes, nil
}

// GetLatestUpdates retrieves the latest anime updates from the source
func (c *CLIScraper) GetLatestUpdates(page int) ([]Anime, error) {
	var animes []Anime

	output, err := c.runCommand("latest", c.SourceID, "--page", strconv.Itoa(page))
	if err != nil {
		return animes, err
	}

	// Convert the data to JSON and then unmarshal to our struct
	data, err := json.Marshal(output.Data)
	if err != nil {
		return animes, fmt.Errorf("failed to re-marshal data: %s", err)
	}

	if err := json.Unmarshal(data, &animes); err != nil {
		return animes, fmt.Errorf("failed to unmarshal anime list: %s", err)
	}

	return animes, nil
}

// SearchAnime searches for anime with the given query and filters
func (c *CLIScraper) SearchAnime(query string, page int, filters string) ([]Anime, error) {
	var animes []Anime

	args := []string{"search", c.SourceID, "--query", query, "--page", strconv.Itoa(page)}
	if filters != "" {
		args = append(args, "--filters", filters)
	}

	output, err := c.runCommand(args...)
	if err != nil {
		return animes, err
	}

	// Convert the data to JSON and then unmarshal to our struct
	data, err := json.Marshal(output.Data)
	if err != nil {
		return animes, fmt.Errorf("failed to re-marshal data: %s", err)
	}

	if err := json.Unmarshal(data, &animes); err != nil {
		return animes, fmt.Errorf("failed to unmarshal anime list: %s", err)
	}

	return animes, nil
}

// GetAnimeDetails retrieves detailed information about an anime
func (c *CLIScraper) GetAnimeDetails(animeID string) (Anime, error) {
	var anime Anime

	output, err := c.runCommand("details", c.SourceID, "--anime", animeID)
	if err != nil {
		return anime, err
	}

	// Convert the data to JSON and then unmarshal to our struct
	data, err := json.Marshal(output.Data)
	if err != nil {
		return anime, fmt.Errorf("failed to re-marshal data: %s", err)
	}

	if err := json.Unmarshal(data, &anime); err != nil {
		return anime, fmt.Errorf("failed to unmarshal anime details: %s", err)
	}

	return anime, nil
}

// GetEpisodeList retrieves the list of episodes for an anime
func (c *CLIScraper) GetEpisodeList(animeID string) ([]Episode, error) {
	var episodes []Episode

	output, err := c.runCommand("episodes", c.SourceID, "--anime", animeID)
	if err != nil {
		return episodes, err
	}

	// Convert the data to JSON and then unmarshal to our struct
	data, err := json.Marshal(output.Data)
	if err != nil {
		return episodes, fmt.Errorf("failed to re-marshal data: %s", err)
	}

	if err := json.Unmarshal(data, &episodes); err != nil {
		return episodes, fmt.Errorf("failed to unmarshal episode list: %s", err)
	}

	return episodes, nil
}

// GetVideoList retrieves stream information for an episode
func (c *CLIScraper) GetVideoList(animeID string, episodeNumber float64) (VideoResponse, error) {
	var response VideoResponse

	output, err := c.runCommand("stream-url", c.SourceID, "--anime", animeID, "--episode", fmt.Sprintf("%g", episodeNumber))
	if err != nil {
		return response, err
	}

	// Convert the data to JSON and then unmarshal to our struct
	data, err := json.Marshal(output.Data)
	if err != nil {
		return response, fmt.Errorf("failed to re-marshal data: %s", err)
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return response, fmt.Errorf("failed to unmarshal video list: %s", err)
	}

	return response, nil
}

// GetMagnetLink retrieves a magnet link for a torrent episode
func (c *CLIScraper) GetMagnetLink(animeID string, episodeNumber float64) (string, error) {
	var response MagnetResponse

	output, err := c.runCommand("magnet-link", c.SourceID, "--anime", animeID, "--episode", fmt.Sprintf("%g", episodeNumber))
	if err != nil {
		return "", err
	}

	// Convert the data to JSON and then unmarshal to our struct
	data, err := json.Marshal(output.Data)
	if err != nil {
		return "", fmt.Errorf("failed to re-marshal data: %s", err)
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal magnet link: %s", err)
	}

	return response.MagnetLink, nil
}

// GetFilterList retrieves the available filters for a source
func (c *CLIScraper) GetFilterList() (FilterResponse, error) {
	var response FilterResponse

	output, err := c.runCommand("filters", c.SourceID)
	if err != nil {
		return response, err
	}

	// Convert the data to JSON and then unmarshal to our struct
	data, err := json.Marshal(output.Data)
	if err != nil {
		return response, fmt.Errorf("failed to re-marshal data: %s", err)
	}

	if err := json.Unmarshal(data, &response); err != nil {
		return response, fmt.Errorf("failed to unmarshal filter list: %s", err)
	}

	return response, nil
}

// GetRelatedAnime retrieves anime related to the given anime
func (c *CLIScraper) GetRelatedAnime(animeID string, page int) ([]Anime, error) {
	var animes []Anime

	output, err := c.runCommand("related", c.SourceID, "--anime", animeID, "--page", strconv.Itoa(page))
	if err != nil {
		return animes, err
	}

	// Convert the data to JSON and then unmarshal to our struct
	data, err := json.Marshal(output.Data)
	if err != nil {
		return animes, fmt.Errorf("failed to re-marshal data: %s", err)
	}

	if err := json.Unmarshal(data, &animes); err != nil {
		return animes, fmt.Errorf("failed to unmarshal related anime list: %s", err)
	}

	return animes, nil
}
