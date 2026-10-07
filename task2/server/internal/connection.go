package internal

import (
	"fmt"
	"net"
)

func ServeConn(conn net.Conn) {
	client := conn.RemoteAddr().String()

	fmt.Printf("client %s: connected", client)

	defer func() {
		if err := conn.Close(); err != nil {
			fmt.Printf("client %s: close connection: %v", client, err)
		}

		fmt.Printf("client %s: disconnected", client)
	}()

	if err := handleConn(conn); err != nil {
		fmt.Printf("client %s: transfer failed: %v", client, err)
		return
	}

	fmt.Printf("client %s: handler completed", client)
}
