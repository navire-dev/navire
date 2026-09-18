package main

import (
	"context"
	"fmt"
	"os"

	navirectl "github.com/navire-dev/navire/navirectl"
)

func main() {
	if err := navirectl.Execute(context.Background(), os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
