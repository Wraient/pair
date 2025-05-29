package scraper

// ScraperInfo represents metadata about a scraper implementation
type ScraperInfo struct {
	ID          string `json:"id"`          // Unique identifier for the scraper
	Name        string `json:"name"`        // Human-readable name
	Version     string `json:"version"`     // Semantic version of the scraper
	Description string `json:"description"` // Brief description of the scraper's functionality
}

// SearchResult represents an anime search result from a scraper
type SearchResult struct {
	ID              string   `json:"id"`                        // Unique identifier for the anime within this scraper
	Title           string   `json:"title"`                     // Primary title of the anime
	AlternateTitles []string `json:"alternateTitles,omitempty"` // Alternative titles (e.g., English, Japanese, etc.)
	TotalEpisodes   int      `json:"totalEpisodes"`             // Total number of episodes available
	Type            string   `json:"type,omitempty"`            // Type of media (TV, Movie, OVA, etc.)
	Year            int      `json:"year,omitempty"`            // Release year
	Status          string   `json:"status,omitempty"`          // Airing status
	Thumbnail       string   `json:"thumbnail,omitempty"`       // URL to thumbnail image
}

// EpisodeInfo represents metadata about a specific episode
type EpisodeInfo struct {
	Number    float64 `json:"number"`              // Episode number (float to support .5 episodes)
	Title     string  `json:"title,omitempty"`     // Episode title if available
	Length    int     `json:"length,omitempty"`    // Episode length in seconds if available
	Thumbnail string  `json:"thumbnail,omitempty"` // Episode thumbnail URL if available
	IsFiller  bool    `json:"isFiller,omitempty"`  // Whether this is a filler episode
}

// StreamInfo represents a stream source for an episode
type StreamInfo struct {
	URL         string            `json:"url"`                   // Direct URL to the video stream
	Quality     string            `json:"quality"`               // Quality label (e.g., "1080p", "720p")
	Format      string            `json:"format"`                // Video format/container (e.g., "mp4", "m3u8")
	Headers     map[string]string `json:"headers,omitempty"`     // Required HTTP headers if any
	SubtitleURL string            `json:"subtitleUrl,omitempty"` // URL to subtitles file if available
}

// ScraperInterface defines the core functionality a scraper must implement
type ScraperInterface interface {
	// GetInfo returns metadata about this scraper implementation
	GetInfo() ScraperInfo

	// Search searches for anime matching the given query
	// mode can be "sub" or "dub" to filter results
	Search(query string, mode string) ([]SearchResult, error)

	// GetEpisodeList retrieves the list of available episodes for an anime
	// mode can be "sub" or "dub" to filter episodes
	GetEpisodeList(animeID string, mode string) ([]EpisodeInfo, error)

	// GetStreamInfo retrieves stream information for a specific episode
	// mode can be "sub" or "dub" for language preference
	GetStreamInfo(animeID string, episodeNumber float64, mode string) ([]StreamInfo, error)
}

// CLIOutput represents the expected JSON output format from scraper CLIs
type CLIOutput struct {
	Status  string      `json:"status"`            // "success" or "error"
	Data    interface{} `json:"data,omitempty"`    // Contains the actual result data
	Error   string      `json:"error,omitempty"`   // Error message if status is "error"
	Message string      `json:"message,omitempty"` // Optional informational message
}
