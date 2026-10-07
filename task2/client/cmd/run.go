package main

import (
	"fmt"
	"os"
	"strconv"
	"task2/client/internal"
)

func run() error {
	if len(os.Args) != 4 {
		return fmt.Errorf(
			"usage: %s <server-host> <server-port> <file-path>",
			os.Args[0],
		)
	}

	host := os.Args[1]
	portArg := os.Args[2]
	filePath := os.Args[3]

	if host == "" {
		return fmt.Errorf("server host is empty")
	}

	port, err := strconv.Atoi(portArg)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf(
			"invalid port %q: expected an integer from 1 to 65535",
			portArg,
		)
	}

	return internal.SendFile(host, port, filePath)
}
