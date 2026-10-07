package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		fmt.Printf("client: %v", err)
		os.Exit(1)
	}
}
