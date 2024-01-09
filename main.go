package main

import (
	"log"

	"github.com/debasishbsws/conxec/pkg/cli"
)

func main() {
	if err := cli.New().Execute(); err != nil {
		log.Fatal(err)
	}
}
