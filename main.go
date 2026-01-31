// Package main implements OpenSeat, a CLI tool for monitoring Virginia Tech
// course sections and notifying users when seats become available.
package main

import (
	"log"
	"os"
)

func main() {
	// Check for --demo flag
	for _, arg := range os.Args[1:] {
		if arg == "--demo" {
			RunDemo()
			return
		}
	}

	if err := Run(RunOptions{ConfigPath: "config.json"}); err != nil {
		log.Fatal(err)
	}
}
