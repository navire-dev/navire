package main

import (
	"fmt"
	"os"

	"github.com/navire-dev/navire/dispatcher/app"
)

func main() {
	instance, err := app.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := instance.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
