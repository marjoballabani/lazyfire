package main

import (
	"fmt"
	"log"
	"os"

	"github.com/mballabani/lazyfire/pkg/app"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	buildInfo := &app.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	}

	app, err := app.NewApp(buildInfo)
	if err != nil {
		log.Fatal(err)
	}

	if err := app.Run(); err != nil {
		if err.Error() != "quit" {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}