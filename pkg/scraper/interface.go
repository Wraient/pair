package scraper

import (
	"context"
	"time"
)

// Scraper defines the interface that all anime scrapers must implement
type Scraper interface {
	// Info returns metadata about the scraper implementation
	Info() ScraperInfo

	// Search searches for anime using the given query and filters
	Search(ctx context.Context, query string, filters SearchFilters) ([]SearchResult, error)

	// GetAnimeInfo retrieves detailed information about a specific anime
	GetAnimeInfo(ctx context.Context, id string) (*AnimeInfo, error)

	// GetEpisodes retrieves the episode list for a specific anime
	GetEpisodes(ctx context.Context, animeID string) ([]EpisodeInfo, error)

	// GetStreams retrieves available video streams for a specific episode
	GetStreams(ctx context.Context, episodeID string) ([]StreamInfo, error)
}

// ScraperInfo contains metadata about a scraper implementation
type ScraperInfo struct {
	ID          string   // Unique identifier for the scraper
	Name        string   // Human-readable name
	Version     string   // Semantic version
	Description string   // Short description
	Website     string   // Associated website
	Languages   []string // Supported subtitle languages
}

// SearchFilters contains optional filters for anime searches
type SearchFilters struct {
	Type     string   // Anime, Movie, OVA, etc.
	Season   string   // Winter, Spring, Summer, Fall
	Year     int      // Release year
	Status   string   // Ongoing, Completed, etc.
	Genres   []string // List of genres to filter by
	Sort     string   // Sorting criteria (popularity, latest, etc.)
	Language string   // Sub/Dub preference
}

// SearchResult represents an anime entry from search results
type SearchResult struct {
	ID             string
	Title          string
	AlternateTitles map[string]string // Map of language codes to titles
	Thumbnail      string
	Type           string
	Year           int
	Status         string
}

// AnimeInfo contains detailed information about an anime
type AnimeInfo struct {
	ID              string
	Title           string
	AlternateTitles map[string]string
	Synopsis        string
	Thumbnail       string
	CoverImage      string
	Type            string
	Episodes        int
	Status          string
	Season          string
	Year           int
	Genres         []string
	Studios        []string
	Score          float64
	Rating         string // Age rating
	Duration       int    // Episode duration in minutes
	AiredFrom      time.Time
	AiredTo        *time.Time // Pointer since it might be ongoing
}

// EpisodeInfo represents a single episode with extended information
type EpisodeInfo struct {
	ID          string
	Number      float64 // Float to support episodes like 13.5
	Title       string
	Thumbnail   string
	ReleaseDate time.Time
	Duration    int // Duration in minutes
	Languages   []string // Available audio languages (sub/dub)
}

// StreamInfo represents a video stream source for an episode
type StreamInfo struct {
	ID       string
	Quality  string // e.g., "1080p", "720p"
	Format   string // e.g., "mp4", "hls"
	URL      string
	Headers  map[string]string // Required HTTP headers for playback
	Type     string // Source type (direct, embedder, etc.)
	Priority int    // Priority for sorting (higher = preferred)

	// Optional fields for specific source types
	SubtitleTracks []SubtitleTrack
	AudioTracks    []AudioTrack
}

// SubtitleTrack represents a subtitle track for a video
type SubtitleTrack struct {
	Language string
	URL      string
	Format   string // e.g., "vtt", "ass"
	Label    string // Optional display label
}

// AudioTrack represents an audio track for a video
type AudioTrack struct {
	Language string
	URL      string
	Format   string // e.g., "aac", "opus"
	Label    string // Optional display label
}

// Status constants
const (
	StatusOngoing     = "ongoing"
	StatusCompleted   = "completed"
	StatusUpcoming    = "upcoming"
	StatusUnknown     = "unknown"
)

// Common video qualities
const (
	Quality2160p = "2160p"
	Quality1440p = "1440p"
	Quality1080p = "1080p"
	Quality720p  = "720p"
	Quality480p  = "480p"
	Quality360p  = "360p"
	Quality240p  = "240p"
)

// Common video formats
const (
	FormatMP4 = "mp4"
	FormatHLS = "hls"
	FormatDASH = "dash"
)