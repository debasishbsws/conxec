package main

import (
	"log"

	"github.com/debasishbsws/conxec/pkg/cmd"
)

var (
	version string = "dev"
	commit  string = "unknown"
)

func main() {
	if err := cmd.New(version, commit).Execute(); err != nil {
		log.Fatal(err)
	}
}
