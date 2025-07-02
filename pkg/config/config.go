package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/wraient/pair/pkg/database"
	"github.com/spf13/viper"
)

var (
	cfg  *Config
	once sync.Once
	db   *database.DB
)

// UIMode represents the UI mode to use
type UIMode string

const (
	UIModeRofi UIMode = "rofi"
	UIModeCLI  UIMode = "cli"
)

// TrackerType represents the anime tracking service to use
type TrackerType string

const (
	TrackerLocal   TrackerType = "local"
	TrackerMAL     TrackerType = "mal"
	TrackerAnilist TrackerType = "anilist"
)

// Config represents the application configuration
type Config struct {
	// UI settings
	UI struct {
		Mode             UIMode `mapstructure:"mode"`
		ShowImagePreview bool   `mapstructure:"show_image_preview"`
		ShowEpPrompt     bool   `mapstructure:"show_episode_prompt"`
	} `mapstructure:"ui"`

	// Anime tracking settings
	Tracking struct {
		Service   TrackerType `mapstructure:"service"`
		AutoSync  bool        `mapstructure:"auto_sync"`
		SyncDelay int         `mapstructure:"sync_delay"` // in minutes
	} `mapstructure:"tracking"`

	// Extension settings
	Extensions struct {
		AutoUpdate bool     `mapstructure:"auto_update"`
		Repos      []string `mapstructure:"repos"`
		Directory  string   `mapstructure:"directory"`
	} `mapstructure:"extensions"`

	// Discord RPC settings
	DiscordRPC struct {
		Enabled       bool `mapstructure:"enabled"`
		ShowProgress  bool `mapstructure:"show_progress"`
		ShowButtons   bool `mapstructure:"show_buttons"`
		ShowTimestamp bool `mapstructure:"show_timestamp"`
	} `mapstructure:"discord_rpc"`

	// Video settings
	Video struct {
		DefaultLanguage string   `mapstructure:"default_language"`
		SubtitleLangs   []string `mapstructure:"subtitle_languages"`
		QualityPrefer   string   `mapstructure:"quality_prefer"`
	} `mapstructure:"video"`

	// API settings
	API struct {
		MALClientID     string `mapstructure:"mal_client_id"`
		AnilistClientID string `mapstructure:"anilist_client_id"`
	} `mapstructure:"api"`

	// Development settings
	Development bool `mapstructure:"development"`

	// Database settings
	DatabaseConfig struct {
		Path string `mapstructure:"path"`
	} `mapstructure:"database"`
}

// Initialize sets up the configuration system
func Initialize() error {
	var initErr error
	once.Do(func() {
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
		viper.AddConfigPath(filepath.Join(os.ExpandEnv("$HOME"), ".config", "pair"))

		// Set defaults
		setDefaults()

		// Create config directory if it doesn't exist
		configDir := filepath.Join(os.ExpandEnv("$HOME"), ".config", "pair")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			initErr = fmt.Errorf("failed to create config directory: %w", err)
			return
		}

		// Create default config file if it doesn't exist
		configFile := filepath.Join(configDir, "config.toml")
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			if err := viper.SafeWriteConfig(); err != nil {
				initErr = fmt.Errorf("failed to write default config: %w", err)
				return
			}
		}

		// Read config
		if err := viper.ReadInConfig(); err != nil {
			initErr = fmt.Errorf("failed to read config: %w", err)
			return
		}

		// Parse config into struct
		cfg = &Config{}
		if err := viper.Unmarshal(cfg); err != nil {
			initErr = fmt.Errorf("failed to parse config: %w", err)
			return
		}

		// Initialize database
		dbPath := viper.GetString("database.path")
		if dbPath == "" {
			dbPath = filepath.Join(os.ExpandEnv("$HOME"), ".local", "share", "pair", "pair.db")
		}

		// Ensure directory exists
		dbDir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			initErr = fmt.Errorf("failed to create database directory: %w", err)
			return
		}

		// Initialize database
		var err error
		db, err = database.New(dbPath)
		if err != nil {
			initErr = fmt.Errorf("failed to initialize database: %w", err)
			return
		}

		// Migrate config values to database
		migrateConfigToDatabase()
	})

	return initErr
}

// migrateConfigToDatabase migrates configuration values from TOML to the database
func migrateConfigToDatabase() {
	// Check if we've already migrated
	migrated, err := db.GetConfig("config_migrated")
	if err == nil && migrated == "true" {
		return
	}

	// Set UI preferences
	db.SetConfig("ui.mode", string(cfg.UI.Mode))
	db.SetConfig("ui.show_image_preview", fmt.Sprintf("%v", cfg.UI.ShowImagePreview))
	db.SetConfig("ui.show_episode_prompt", fmt.Sprintf("%v", cfg.UI.ShowEpPrompt))

	// Set tracking preferences
	db.SetConfig("tracking.service", string(cfg.Tracking.Service))
	db.SetConfig("tracking.auto_sync", fmt.Sprintf("%v", cfg.Tracking.AutoSync))
	db.SetConfig("tracking.sync_delay", fmt.Sprintf("%d", cfg.Tracking.SyncDelay))

	// Set video preferences
	db.SetConfig("video.default_language", cfg.Video.DefaultLanguage)
	db.SetConfig("video.quality_prefer", cfg.Video.QualityPrefer)

	// Set API keys
	if cfg.API.MALClientID != "" {
		db.SetConfig("api.mal_client_id", cfg.API.MALClientID)
	}
	if cfg.API.AnilistClientID != "" {
		db.SetConfig("api.anilist_client_id", cfg.API.AnilistClientID)
	}

	// Mark as migrated
	db.SetConfig("config_migrated", "true")
}

// setDefaults sets the default configuration values
func setDefaults() {
	viper.SetDefault("ui.mode", UIModeRofi)
	viper.SetDefault("ui.show_image_preview", true)
	viper.SetDefault("ui.show_episode_prompt", true)

	viper.SetDefault("tracking.service", TrackerLocal)
	viper.SetDefault("tracking.auto_sync", true)
	viper.SetDefault("tracking.sync_delay", 30)

	viper.SetDefault("extensions.auto_update", true)
	viper.SetDefault("extensions.repos", []string{})

	viper.SetDefault("discord_rpc.enabled", true)
	viper.SetDefault("discord_rpc.show_progress", true)
	viper.SetDefault("discord_rpc.show_buttons", true)
	viper.SetDefault("discord_rpc.show_timestamp", true)

	viper.SetDefault("video.default_language", "en")
	viper.SetDefault("video.subtitle_languages", []string{"en"})
	viper.SetDefault("video.quality_prefer", "1080p")

	viper.SetDefault("extensions.directory", filepath.Join(os.ExpandEnv("$HOME"), ".local", "share", "pair", "extensions"))

	viper.SetDefault("development", false)

	// Database settings
	viper.SetDefault("database.path", filepath.Join(os.ExpandEnv("$HOME"), ".local", "share", "pair", "pair.db"))
}

// Get returns the current configuration
func Get() *Config {
	if cfg == nil {
		panic("config not initialized")
	}
	return cfg
}

// GetDB returns the database connection
func GetDB() *database.DB {
	if db == nil {
		panic("database not initialized")
	}
	return db
}

// GetConfigDir returns the configuration directory
func GetConfigDir() string {
	return filepath.Join(os.ExpandEnv("$HOME"), ".config", "pair")
}

// Save writes the current configuration to disk
func Save() error {
	for k, v := range viper.AllSettings() {
		viper.Set(k, v)
	}
	return viper.WriteConfig()
}