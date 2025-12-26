package main

import (
	"fmt"
	"os"

	"github.com/schmoli/macos-setup/internal/tui"
)

func main() {
	if err := tui.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
