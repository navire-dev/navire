package main

import (
	"fmt"
	"os"

	"github.com/navire-dev/navire/core/app"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	newApp, err := app.New()
	if err != nil {
		return err
	}

	return newApp.Run()
}
