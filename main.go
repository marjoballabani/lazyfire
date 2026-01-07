// LazyFire is a terminal UI for browsing Firebase Firestore.
// It provides a lazygit-inspired interface for navigating collections and documents.
//
// Usage:
//
//	lazyfire
//
// Configuration is loaded from ~/.lazyfire/config.yaml
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/marjoballabani/lazyfire/pkg/app"
)

// Build information, set via ldflags during compilation:
//
//	go build -ldflags "-X main.version=1.0.0 -X main.commit=$(git rev-parse HEAD)"
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Handle --version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("lazyfire %s\n", version)
		return
	}

	buildInfo := &app.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	}

	application, err := app.NewApp(buildInfo)
	if err != nil {
		log.Fatal(err)
	}

	if err := application.Run(); err != nil {
		// "quit" is a normal exit via 'q' key
		if err.Error() != "quit" {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}
