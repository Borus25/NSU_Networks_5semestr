package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"task2/server/internal"
)

func main() {
	if len(os.Args) != 2 {
		log.Println("Usage: ./server <server_port>")
		os.Exit(1)
	}
	port := os.Args[1]

	err := os.MkdirAll("uploads", os.FileMode(0755))
	if err != nil {
		_ = fmt.Errorf("os.MkdirAll error: %w", err)
		os.Exit(1)
	}

	ls, err := net.Listen("tcp", ":"+port)
	if err != nil {
		err = fmt.Errorf("net.Listen error: %w", err)
		log.Print(err.Error())
		os.Exit(1)
	}
	defer func(ls net.Listener) {
		errLs := ls.Close()
		if errLs != nil {
			errLs = fmt.Errorf("net.Listener.Close error: %w", err)
			log.Print(errLs.Error())

		}
	}(ls)

	for {
		conn, errConn := ls.Accept()
		if errConn != nil {
			errConn = fmt.Errorf("accept connection: %w", errConn)
			log.Print(errConn.Error())

			continue
		}

		go internal.ServeConn(conn)
	}
}
