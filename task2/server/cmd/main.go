package main

import (
	"log"
	"os"

	"task2/server/internal"
)

func main() {
	log.SetFlags(0)

	if err := internal.Run(os.Args); err != nil {
		log.Printf("server: %v", err)
		os.Exit(1)
	}
}
