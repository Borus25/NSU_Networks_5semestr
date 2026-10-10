package internal

import (
	"log"
	"net"
)

func ServeConn(conn net.Conn) {
	client := conn.RemoteAddr().String()

	log.Printf("client %s: connected", client)

	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("client %s: close connection: %v", client, err)
		}

		log.Printf("client %s: disconnected", client)
	}()

	if err := handleConn(conn); err != nil {
		log.Printf("client %s: transfer failed: %v", client, err)
		return
	}

	log.Printf("client %s: handler completed", client)
}
