package appcore

import (
	"context"

	"github.com/wraient/pair/pkg/config"
	"github.com/wraient/pair/pkg/tracker"
	"github.com/wraient/pair/pkg/ui"
)

// App represents the main application
type App struct {
	ctx         context.Context
	menuManager *ui.MenuManager
	trackerMgr  *tracker.TrackerManager
	currentMenu *ui.Menu
	config      *config.Config
}

// NewApp creates a new App instance
func NewApp(ctx context.Context) *App {
	return &App{
		ctx:         ctx,
		menuManager: ui.NewMenuManager(ctx),
		config:      config.Get(),
		trackerMgr:  tracker.NewTrackerManager(config.GetDB()),
	}
}

// Start starts the application
func Start() error {
	app := NewApp(context.Background())

	// Register trackers
	anilistTracker := tracker.NewAnilistTracker(config.GetConfigDir())
	app.trackerMgr.RegisterTracker(anilistTracker)

	// Setup main menu
	mainMenu := app.setupMainMenu()

	// Start menu loop
	return app.menuManager.Show(mainMenu)
}
