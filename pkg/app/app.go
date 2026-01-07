// Package app is the main application entry point for LazyFire.
// It coordinates initialization of configuration, Firebase client, and the GUI.
package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/marjoballabani/lazyfire/pkg/config"
	"github.com/marjoballabani/lazyfire/pkg/firebase"
	"github.com/marjoballabani/lazyfire/pkg/gui"
	"github.com/pkg/errors"
)

// BuildInfo contains version information set at compile time.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// App is the main application struct that holds all components.
type App struct {
	buildInfo      *BuildInfo
	config         *config.Config
	firebaseClient *firebase.Client
	gui            *gui.Gui
	ctx            context.Context
}

// NewApp creates a new App instance with the given build information.
// It loads configuration but does not initialize Firebase or GUI yet.
func NewApp(buildInfo *BuildInfo) (*App, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load config")
	}

	return &App{
		buildInfo: buildInfo,
		config:    cfg,
		ctx:       context.Background(),
	}, nil
}

// Run starts the application by initializing Firebase, creating the GUI,
// and running the main event loop. It blocks until the user quits.
func (app *App) Run() error {
	// Initialize Firebase client using existing auth credentials
	firebaseClient, err := firebase.NewClient(app.ctx, app.config)
	if err != nil {
		// Provide helpful error message for authentication issues
		if strings.Contains(err.Error(), "no authentication found") {
			fmt.Println("\n🔐 Authentication Required")
			fmt.Println("\nLazyFire needs you to be authenticated with Firebase or Google Cloud.")
			fmt.Println("\nPlease run one of the following commands:")
			fmt.Println("  • firebase login              (recommended)")
			fmt.Println("  • gcloud auth application-default login")
			fmt.Println("\nAfter logging in, run lazyfire again.")
			return fmt.Errorf("authentication required")
		}
		return errors.Wrap(err, "failed to initialize Firebase client")
	}
	app.firebaseClient = firebaseClient

	// Initialize and run the terminal UI
	gui, err := gui.NewGui(app.config, app.firebaseClient, app.buildInfo.Version)
	if err != nil {
		return errors.Wrap(err, "failed to initialize GUI")
	}
	app.gui = gui

	return app.gui.Run()
}
