// Package main implements a CLI tool for monitoring Virginia Tech course sections
// and notifying users when seats become available.
package main

import (
	"flag"
	"log"
)

func main() {
	configPath := flag.String("config", "config.json", "Path to config file")
	flag.Parse()

	if err := Run(*configPath); err != nil {
		log.Fatal(err)
	}
}
