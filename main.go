package main

import (
	"log"

	"github.com/debasishbsws/conxec/pkg/cmd"
)

func main() {
	if err := cmd.New().Execute(); err != nil {
		log.Fatal(err)
	}
}
